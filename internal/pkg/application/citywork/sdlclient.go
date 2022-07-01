package citywork

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

//go:generate moq -rm -out sdlclient_mock.go . SdlClient

var sdltracer = otel.Tracer("sdl-trafficinfo-client")

type SdlClient interface {
	Get(cxt context.Context) (*sdlResponse, error)
}

type sdlClient struct {
	sundsvallvaxerURL string
}

func NewSdlClient(sundsvallvaxerURL string, log zerolog.Logger) SdlClient {
	return &sdlClient{
		sundsvallvaxerURL: sundsvallvaxerURL,
	}
}

func (c *sdlClient) Get(ctx context.Context) (*sdlResponse, error) {
	var err error
	ctx, span := sdltracer.Start(ctx, "get-sdl-traffic-information")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	log := logging.GetFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.sundsvallvaxerURL, nil)
	if err != nil {
		return nil, err
	}

	apiResponse, err := httpClient.Do(apiReq)
	if err != nil {
		log.Error().Err(err).Msg("failed to retrieve traffic information")
		return nil, err
	}

	defer apiResponse.Body.Close()

	if apiResponse.StatusCode != http.StatusOK {
		log.Error().Msgf("failed to retrieve traffic information, expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
		return nil, fmt.Errorf("expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
	}

	body, err := io.ReadAll(apiResponse.Body)
	if err != nil {
		log.Error().Err(err).Msg("failed to read response body")
		return nil, err
	}

	log.Info().Msgf("received response: %s...", string(body)[:100])

	var m sdlResponse
	err = json.Unmarshal(body, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal model")
	}

	return &m, err
}
