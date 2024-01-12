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
	apiKey    string
	sourceURL string
}

func NewClient(apikey, sourceURL string) Client {
	return &clientImpl{
		apiKey:    apikey,
		sourceURL: sourceURL,
	}
}

func (c *clientImpl) Get(ctx context.Context) (*domain.FeatureCollection, error) {
	var err error
	ctx, span := sdltracer.Start(ctx, "get-facilities-information")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	log := logging.GetFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.sourceURL+"/list", nil)
	if err != nil {
		return nil, err
	}

	apiReq.Header.Set("apikey", c.apiKey)

	apiResponse, err := httpClient.Do(apiReq)
	if err != nil {
		log.Error("failed to retrieve facilities information", "err", err.Error())
		return nil, err
	}
	defer apiResponse.Body.Close()

	if apiResponse.StatusCode != http.StatusOK {
		log.Error("failed to retrieve facilities information", slog.Int("statusCode", apiResponse.StatusCode), "err", err.Error())
		return nil, fmt.Errorf("expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
	}

	body, err := io.ReadAll(apiResponse.Body)
	if err != nil {
		log.Error("failed to read response body", "err", err.Error())
		return nil, err
	}

	featureCollection := &domain.FeatureCollection{}
	err = json.Unmarshal(body, featureCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response from %s. (%s)", c.sourceURL, err.Error())
	}

	return featureCollection, nil
}
