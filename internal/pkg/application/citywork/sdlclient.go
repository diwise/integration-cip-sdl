package citywork

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

//go:generate moq -rm -out sdlclient_mock.go . SdlClient

var sdltracer = otel.Tracer("sdl-cityworks-client")

type SdlClient interface {
	Get(cxt context.Context) (*sdlResponse, error)
}

type sdlClient struct {
	sundsvallvaxerURL string
	httpClient        http.Client
}

func NewSdlClient(ctx context.Context, sundsvallvaxerURL string) SdlClient {
	return &sdlClient{
		sundsvallvaxerURL: sundsvallvaxerURL,
		httpClient: http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

func (c *sdlClient) Get(ctx context.Context) (*sdlResponse, error) {
	var err error
	ctx, span := sdltracer.Start(ctx, "get-sdl-cityworks-info")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	log := logging.GetFromContext(ctx)

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.sundsvallvaxerURL, nil)
	if err != nil {
		return nil, err
	}

	apiResponse, err := c.httpClient.Do(apiReq)
	if err != nil {
		log.Error("failed to retrieve citywork information", "err", err.Error())
		return nil, err
	}

	defer apiResponse.Body.Close()

	if apiResponse.StatusCode != http.StatusOK {
		log.Error("unexpected response code when retrieving traffic information", slog.Int("expected", http.StatusOK), slog.Int("received", apiResponse.StatusCode))
		return nil, fmt.Errorf("expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
	}

	body, err := io.ReadAll(apiResponse.Body)
	if err != nil {
		log.Error("failed to read response body", "err", err.Error())
		return nil, err
	}

	strBody := string(body)
	if len(strBody) > 100 {
		strBody = strBody[:100]
	}

	log.Debug("received response", "body", strBody)

	var m sdlResponse
	err = json.Unmarshal(body, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal model")
	}

	if len(m.Error) > 0 {
		return nil, fmt.Errorf("endpoint returned 200 OK with err body: (%s)", m.Error)
	}

	return &m, err
}
