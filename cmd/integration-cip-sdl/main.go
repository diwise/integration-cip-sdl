package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/integration-cip-sdl/internal/domain"
	"github.com/diwise/integration-cip-sdl/internal/pkg/application/citywork"
	"github.com/diwise/integration-cip-sdl/internal/pkg/application/facilities"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/go-chi/chi"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
)

func main() {
	serviceVersion := buildinfo.SourceVersion()
	serviceName := "integration-cip-sdl"

	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	contextBrokerURL := env.GetVariableOrDie(logger, "CONTEXT_BROKER_URL", "Context Broker URL")
	ctxBroker := domain.NewContextBrokerClient(contextBrokerURL, logger)

	if featureIsEnabled(logger, "facilities") {
		facilitiesURL := env.GetVariableOrDie(logger, "FACILITIES_URL", "Facilities URL")
		facilitiesApiKey := env.GetVariableOrDie(logger, "FACILITIES_API_KEY", "Facilities Api Key")
		timeInterval := env.GetVariableOrDefault(logger, "FACILITIES_POLLING_INTERVAL", "60")

		parsedTime, err := strconv.ParseInt(timeInterval, 0, 64)
		if err != nil {
			logger.Fatal().Err(err).Msg("FACILITIES_POLLING_INTERVAL must be set to valid integer")
		}

		go SetupAndRunFacilities(facilitiesURL, facilitiesApiKey, int(parsedTime), logger, ctx, ctxBroker)
	}

	if featureIsEnabled(logger, "citywork") {
		sundsvallvaxerURL := env.GetVariableOrDie(logger, "SDL_KARTA_URL", "Sundsvall växer URL")
		cw := SetupCityWorkService(logger, sundsvallvaxerURL, ctxBroker)
		go cw.Start(ctx)
	}

	setupRouterAndWaitForConnections(logger)
}

//featureIsEnabled checks wether a given feature is enabled by exanding the feature name into <uppercase>_ENABLED and checking if the corresponding environment variable is set to true.
//  Ex: citywork -> CITYWORK_ENABLED
func featureIsEnabled(logger zerolog.Logger, feature string) bool {
	featureKey := fmt.Sprintf("%s_ENABLED", strings.ToUpper(feature))
	isEnabled := os.Getenv(featureKey) == "true"

	if isEnabled {
		logger.Info().Msgf("feature %s is enabled", feature)
	} else {
		logger.Warn().Msgf("feature %s is not enabled", feature)
	}

	return isEnabled
}

func SetupCityWorkService(log zerolog.Logger, sundsvallvaxerURL string, ctxBroker domain.ContextBrokerClient) citywork.CityWorkSvc {
	c := citywork.NewSdlClient(sundsvallvaxerURL, log)

	return citywork.NewCityWorkService(log, c, ctxBroker)
}

func setupRouterAndWaitForConnections(logger zerolog.Logger) {
	r := chi.NewRouter()
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to start router")
	}
}

func SetupAndRunFacilities(url, apiKey string, timeInterval int, logger zerolog.Logger, ctx context.Context, ctxBroker domain.ContextBrokerClient) facilities.Client {

	fc := facilities.NewFacilitiesClient(apiKey, url, logger)

	for {
		features, err := fc.Get(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("failed to retrieve facilities information")
		}

		err = facilities.StoreTrailsFromSource(logger, ctxBroker, ctx, url, *features)
		if err != nil {
			logger.Error().Err(err).Msg("failed to store exercise trails information")
		}
		err = facilities.StoreBeachesFromSource(logger, ctxBroker, ctx, url, *features)
		if err != nil {
			logger.Error().Err(err).Msg("failed to store beaches information")
		}

		time.Sleep(time.Duration(timeInterval) * time.Minute)

	}
}
