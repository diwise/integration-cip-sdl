package citywork

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/diwise/context-broker/pkg/datamodels/fiware"
	"github.com/diwise/context-broker/pkg/ngsild/client"
	ngsitypes "github.com/diwise/context-broker/pkg/ngsild/types"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/rs/zerolog"
)

type CityWorkSvc interface {
	Start(ctx context.Context) error
	//TODO: This is not supposed to be a public interface (only exposed for testing it seems)
	getAndPublishCityWork(ctx context.Context) error
}

func NewCityWorkService(log zerolog.Logger, s SdlClient, timeInterval int, c client.ContextBrokerClient) CityWorkSvc {
	return &cw{
		log:           log,
		sdlClient:     s,
		timeInterval:  timeInterval,
		contextbroker: c,
	}
}

type cw struct {
	log           zerolog.Logger
	sdlClient     SdlClient
	timeInterval  int
	contextbroker client.ContextBrokerClient
}

var previous map[string]string = make(map[string]string)

func (cw *cw) Start(ctx context.Context) error {
	for {
		err := cw.getAndPublishCityWork(ctx)
		sleepDuration := time.Duration(cw.timeInterval) * time.Minute

		if err != nil {
			const retryIntervalMinutes int = 2
			cw.log.Error().Err(err).Msgf("failed to get city work, attempting again in %d minutes", retryIntervalMinutes)
			sleepDuration = time.Duration(retryIntervalMinutes) * time.Minute
		}

		time.Sleep(sleepDuration)
	}
}

func (cw *cw) getAndPublishCityWork(ctx context.Context) error {
	response, err := cw.sdlClient.Get(ctx)
	if err != nil {
		cw.log.Error().Err(err).Msg("failed to get city work")
		return fmt.Errorf("failed to get city work")
	}

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	for _, f := range response.Features {
		featureID := f.ID()
		if _, exists := previous[featureID]; exists {
			continue
		}

		cwModel := toCityWorkModel(f)

		_, err = cw.contextbroker.CreateEntity(ctx, cwModel, headers)
		if err != nil {
			cw.log.Error().Err(err).Msg("failed to add entity")
			continue
		}

		previous[featureID] = featureID
	}

	return nil
}

func toCityWorkModel(sf sdlFeature) ngsitypes.Entity {
	long, lat, _ := sf.Geometry.AsPoint()

	startDate := strings.ReplaceAll(sf.Properties.Start, "Z", "") + "T00:00:00Z"
	endDate := strings.ReplaceAll(sf.Properties.End, "Z", "") + "T23:59:59Z"

	cw, _ := entities.New(fiware.CityWorkIDPrefix+sf.ID(), fiware.CityWorkTypeName,
		decorators.Location(lat, long),
		decorators.Description(sf.Properties.Description),
		decorators.DateTime("startDate", startDate),
		decorators.DateTime("endDate", endDate),
		decorators.DateTime("dateCreated", time.Now().UTC().Format(time.RFC3339)),
	)

	return cw
}
