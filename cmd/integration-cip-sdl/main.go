package main

import (
	"context"
	"fmt"
	"log/slog"
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
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/go-chi/chi"
	"github.com/rs/cors"
)

const serviceName string = "integration-cip-sdl"

func main() {
	serviceVersion := buildinfo.SourceVersion()

	ctx, _, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	contextBrokerURL := env.GetVariableOrDie(ctx, "CONTEXT_BROKER_URL", "Context Broker URL")

	ctxBroker := client.NewContextBrokerClient(contextBrokerURL, client.Debug("true"))

	if featureIsEnabled(ctx, "facilities") {
		facilitiesURL := env.GetVariableOrDie(ctx, "FACILITIES_URL", "Facilities URL")
		facilitiesApiKey := env.GetVariableOrDie(ctx, "FACILITIES_API_KEY", "Facilities Api Key")
		timeInterval := env.GetVariableOrDefault(ctx, "FACILITIES_POLLING_INTERVAL", "58")

		parsedTime, err := strconv.ParseInt(timeInterval, 0, 64)
		if err != nil {
			fatal(ctx, "FACILITIES_POLLING_INTERVAL must be set to a valid integer", err)
		}

		go SetupAndRunFacilities(ctx, facilitiesURL, facilitiesApiKey, int(parsedTime), ctxBroker)
	}

	if featureIsEnabled(ctx, "citywork") {
		sundsvallvaxerURL := env.GetVariableOrDie(ctx, "SDL_KARTA_URL", "Sundsvall växer URL")
		timeInterval := env.GetVariableOrDefault(ctx, "CITYWORK_POLLING_INTERVAL", "59")

		parsedTime, err := strconv.ParseInt(timeInterval, 0, 64)
		if err != nil {
			fatal(ctx, "CITYWORK_POLLING_INTERVAL must be set to valid integer", err)
		}

		cw := SetupCityWorkService(ctx, sundsvallvaxerURL, int(parsedTime), ctxBroker)
		go cw.Start(ctx)
	}

	port := env.GetVariableOrDefault(ctx, "SERVICE_PORT", "8080")

	setupRouterAndWaitForConnections(ctx, port)
}

// featureIsEnabled checks wether a given feature is enabled by exanding the feature name into <uppercase>_ENABLED
// and checking if the corresponding environment variable is set to true.
//
//	Ex: citywork -> CITYWORK_ENABLED
func featureIsEnabled(ctx context.Context, feature string) bool {
	featureKey := fmt.Sprintf("%s_ENABLED", strings.ToUpper(feature))
	isEnabled := os.Getenv(featureKey) == "true"

	logger := logging.GetFromContext(ctx)

	if isEnabled {
		logger.Info(fmt.Sprintf("feature %s is enabled", feature))
	} else {
		logger.Warn(fmt.Sprintf("feature %s is enabled", feature))
	}

	return isEnabled
}

func SetupCityWorkService(ctx context.Context, cityWorkURL string, timeInterval int, ctxBroker client.ContextBrokerClient) citywork.CityWorkSvc {
	c := citywork.NewSdlClient(ctx, cityWorkURL)

	return citywork.NewCityWorkService(ctx, c, timeInterval, ctxBroker)
}

func setupRouterAndWaitForConnections(ctx context.Context, port string) {
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
		fatal(ctx, "failed to start router", err)
	}
}

func SetupAndRunFacilities(ctx context.Context, url, apiKey string, timeInterval int, ctxBroker client.ContextBrokerClient) facilities.Client {

	fc := facilities.NewClient(ctx, apiKey, url)
	storage := facilities.NewStorage(ctx)

	logger := logging.GetFromContext(ctx)

	for {
		features, err := fc.Get(ctx)
		sleepDuration := time.Duration(timeInterval) * time.Minute

		if err != nil {
			const retryInterval int = 2
			logger.Error("failed to retrieve facilities information", slog.Int("retry_in", retryInterval), "err", err.Error())
			sleepDuration = time.Duration(retryInterval) * time.Minute
		} else {
			err = storage.StoreTrailsFromSource(ctx, ctxBroker, url, *features)
			if err != nil {
				logger.Error("failed to store exercise trails information", "err", err.Error())
			}
			err = storage.StoreBeachesFromSource(ctx, ctxBroker, url, *features)
			if err != nil {
				logger.Error("failed to store beaches information", "err", err.Error())
			}
			err = storage.StoreSportsFieldsFromSource(ctx, ctxBroker, url, *features)
			if err != nil {
				logger.Error("failed to store sports fields information", "err", err.Error())
			}
			err = storage.StoreSportsVenuesFromSource(ctx, ctxBroker, url, *features)
			if err != nil {
				logger.Error("failed to store sports venues information", "err", err.Error())
			}
		}

		time.Sleep(sleepDuration)
	}
}

func fatal(ctx context.Context, msg string, err error) {
	logger := logging.GetFromContext(ctx)
	logger.Error(msg, "err", err.Error())
	os.Exit(1)
}
