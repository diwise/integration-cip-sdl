package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/diwise/integration-cip-sdl/internal/pkg/application/citywork"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/diwise/integration-cip-sdl/internal/pkg/infrastructure/logging"
	"github.com/diwise/integration-cip-sdl/internal/pkg/infrastructure/tracing"
	"github.com/rs/zerolog"
)

func main() {
	serviceVersion := version()
	serviceName := "integration-cip-sdl"

	ctx, logger := logging.NewLogger(context.Background(), serviceName, serviceVersion)
	logger.Info().Msg("starting up ...")

	cleanup, err := tracing.Init(ctx, logger, serviceName, serviceVersion)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init tracing")
	}
	defer cleanup()

	contextBrokerURL := getEnvironmentVariableOrDie(logger, "CONTEXT_BROKER_URL", "Context Broker URL")

	if featureIsEnabled(logger, "citywork") {
		sundsvallvaxerURL := getEnvironmentVariableOrDie(logger, "SDL_KARTA_URL", "Sundsvall växer URL")
		cw := SetupCityWorkService(logger, sundsvallvaxerURL, contextBrokerURL)
		go cw.Start(ctx)
	}

	for {
		time.Sleep(5 * time.Second)
	}
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

func SetupCityWorkService(log zerolog.Logger, sundsvallvaxerURL string, contextBrokerUrl string) citywork.CityWorkSvc {
	c := citywork.NewSdlClient(sundsvallvaxerURL, log)
	b := domain.NewContextBrokerClient(contextBrokerUrl, log)

	return citywork.NewCityWorkService(log, c, b)
}

func version() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	buildSettings := buildInfo.Settings
	infoMap := map[string]string{}
	for _, s := range buildSettings {
		infoMap[s.Key] = s.Value
	}

	sha := infoMap["vcs.revision"]
	if infoMap["vcs.modified"] == "true" {
		sha += "+"
	}

	return sha
}

func getEnvironmentVariableOrDie(log zerolog.Logger, envVar, description string) string {
	value := os.Getenv(envVar)
	if value == "" {
		log.Fatal().Msgf("Please set %s to a valid %s.", envVar, description)
	}
	return value
}
