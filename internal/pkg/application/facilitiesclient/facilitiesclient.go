package facilities

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var sdltracer = otel.Tracer("facilities-client")

type Client interface {
	Get(cxt context.Context) ([]byte, error)
}

type client struct {
	apiKey    string
	sourceURL string
}

func NewFacilitiesClient(apikey, sourceURL string, log zerolog.Logger) Client {
	return &client{
		apiKey:    apikey,
		sourceURL: sourceURL,
	}
}

func (c *client) Get(ctx context.Context) ([]byte, error) {
	var err error
	ctx, span := sdltracer.Start(ctx, "get-facilities-information")
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

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.sourceURL, nil)
	if err != nil {
		return nil, err
	}

	apiReq.Header.Set("apikey", c.apiKey)

	apiResponse, err := httpClient.Do(apiReq)
	if err != nil {
		log.Error().Err(err).Msgf("failed to retrieve facilities information")
		return nil, err
	}

	if apiResponse.StatusCode != http.StatusOK {
		log.Error().Msgf("failed to retrieve facilities information, expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
		return nil, fmt.Errorf("expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
	}

	defer apiResponse.Body.Close()

	body, err := io.ReadAll(apiResponse.Body)
	if err != nil {
		log.Error().Err(err).Msgf("failed to read response body")
		return nil, err
	}

	return body, nil
}
