package facilities

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var sdltracer = otel.Tracer("facilities-client")

type Client interface {
	Get(ctx context.Context) (*domain.FeatureCollection, error)
}

type clientImpl struct {
	apiKey     string
	sourceURL  string
	httpClient http.Client
}

func NewClient(ctx context.Context, apikey, sourceURL string) Client {
	return &clientImpl{
		apiKey:    apikey,
		sourceURL: sourceURL,
		httpClient: http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

func (c *clientImpl) Get(ctx context.Context) (*domain.FeatureCollection, error) {
	var err error
	ctx, span := sdltracer.Start(ctx, "get-facilities-information")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	log := logging.GetFromContext(ctx)

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.sourceURL+"/list", nil)
	if err != nil {
		return nil, err
	}

	apiReq.Header.Set("apikey", c.apiKey)

	apiResponse, err := c.httpClient.Do(apiReq)
	if err != nil {
		log.Error("failed to retrieve facilities information", "err", err.Error())
		return nil, err
	}
	defer apiResponse.Body.Close()

	if apiResponse.StatusCode != http.StatusOK {
		log.Error("unexpected status code when attempting to retrieve facilities information", slog.Int("expected", http.StatusOK), slog.Int("received", apiResponse.StatusCode))
		err = fmt.Errorf("expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
		return nil, err
	}

	body, err := io.ReadAll(apiResponse.Body)
	if err != nil {
		log.Error("failed to read response body", "err", err.Error())
		return nil, err
	}

	featureCollection := &domain.FeatureCollection{}
	err = json.Unmarshal(body, featureCollection)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal response from %s. (%s)", c.sourceURL, err.Error())
		return nil, err
	}

	return featureCollection, nil
}
