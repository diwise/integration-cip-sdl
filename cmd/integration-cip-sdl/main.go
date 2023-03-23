package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/integration-cip-sdl/internal/pkg/application/citywork"
	"github.com/diwise/integration-cip-sdl/internal/pkg/application/facilities"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/go-chi/chi"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
)

const serviceName string = "integration-cip-sdl"

func main() {
	serviceVersion := buildinfo.SourceVersion()

	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	contextBrokerURL := env.GetVariableOrDie(logger, "CONTEXT_BROKER_URL", "Context Broker URL")

	ctxBroker := client.NewContextBrokerClient(contextBrokerURL, client.Debug("true"))

	if featureIsEnabled(logger, "facilities") {
		facilitiesURL := env.GetVariableOrDie(logger, "FACILITIES_URL", "Facilities URL")
		facilitiesApiKey := env.GetVariableOrDie(logger, "FACILITIES_API_KEY", "Facilities Api Key")
		timeInterval := env.GetVariableOrDefault(logger, "FACILITIES_POLLING_INTERVAL", "58")

		parsedTime, err := strconv.ParseInt(timeInterval, 0, 64)
		if err != nil {
			logger.Fatal().Err(err).Msg("FACILITIES_POLLING_INTERVAL must be set to valid integer")
		}

		go SetupAndRunFacilities(facilitiesURL, facilitiesApiKey, int(parsedTime), logger, ctx, ctxBroker)
	}

	if featureIsEnabled(logger, "citywork") {
		sundsvallvaxerURL := env.GetVariableOrDie(logger, "SDL_KARTA_URL", "Sundsvall växer URL")
		timeInterval := env.GetVariableOrDefault(logger, "CITYWORK_POLLING_INTERVAL", "59")

		parsedTime, err := strconv.ParseInt(timeInterval, 0, 64)
		if err != nil {
			logger.Fatal().Err(err).Msg("CITYWORK_POLLING_INTERVAL must be set to valid integer")
		}

		cw := SetupCityWorkService(logger, sundsvallvaxerURL, int(parsedTime), ctxBroker)
		go cw.Start(ctx)
	}

	port := env.GetVariableOrDefault(logger, "SERVICE_PORT", "8080")

	setupRouterAndWaitForConnections(logger, port)
}

// featureIsEnabled checks wether a given feature is enabled by exanding the feature name into <uppercase>_ENABLED
// and checking if the corresponding environment variable is set to true.
//
//	Ex: citywork -> CITYWORK_ENABLED
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

func SetupCityWorkService(log zerolog.Logger, cityWorkURL string, timeInterval int, ctxBroker client.ContextBrokerClient) citywork.CityWorkSvc {
	c := citywork.NewSdlClient(cityWorkURL, log)

	return citywork.NewCityWorkService(log, c, timeInterval, ctxBroker)
}

func setupRouterAndWaitForConnections(logger zerolog.Logger, port string) {
	r := chi.NewRouter()
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to start router")
	}
}

func SetupAndRunFacilities(url, apiKey string, timeInterval int, logger zerolog.Logger, ctx context.Context, ctxBroker client.ContextBrokerClient) facilities.Client {

	fc := facilities.NewClient(apiKey, url, logger)
	storage := facilities.NewStorage(ctx)

	for {
		features, err := fc.Get(ctx)
		sleepDuration := time.Duration(timeInterval) * time.Minute

		if err != nil {
			const retryInterval int = 2
			logger.Error().Err(err).Msgf("failed to retrieve facilities information (retrying in %d minutes)", retryInterval)
			sleepDuration = time.Duration(retryInterval) * time.Minute
		} else {
			err = storage.StoreTrailsFromSource(ctx, ctxBroker, url, *features)
			if err != nil {
				logger.Error().Err(err).Msg("failed to store exercise trails information")
			}
			err = storage.StoreBeachesFromSource(ctx, ctxBroker, url, *features)
			if err != nil {
				logger.Error().Err(err).Msg("failed to store beaches information")
			}
			err = storage.StoreSportsFieldsFromSource(ctx, ctxBroker, url, *features)
			if err != nil {
				logger.Error().Err(err).Msg("failed to store sports fields information")
			}
			err = storage.StoreSportsVenuesFromSource(ctx, ctxBroker, url, *features)
			if err != nil {
				logger.Error().Err(err).Msg("failed to store sports venues information")
			}
		}

		time.Sleep(sleepDuration)
	}
}
