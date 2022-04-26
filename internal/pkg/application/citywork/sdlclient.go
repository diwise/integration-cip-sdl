package citywork

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

//go:generate moq -rm -out sdlclient_mock.go . SdlClient

var sdltracer = otel.Tracer("sdl-trafficinfo-client")

type SdlClient interface {
	Get(cxt context.Context) ([]byte, error)
}

type sdlClient struct {
	sundsvallvaxerURL string
}

func NewSdlClient(sundsvallvaxerURL string, log zerolog.Logger) SdlClient {
	return &sdlClient{
		sundsvallvaxerURL: sundsvallvaxerURL,
	}
}

func (c *sdlClient) Get(ctx context.Context) ([]byte, error) {
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
		log.Error().Msgf("failed to retrieve traffic information")
		return nil, err
	}
	if apiResponse.StatusCode != http.StatusOK {
		log.Error().Msgf("failed to retrieve traffic information, expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
		return nil, errors.New("")
	}

	defer apiResponse.Body.Close()

	responseBody, err := ioutil.ReadAll(apiResponse.Body)

	log.Info().Msgf("received response: " + string(responseBody))

	return responseBody, err
}
