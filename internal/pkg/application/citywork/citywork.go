package citywork

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/diwise/context-broker/pkg/datamodels/fiware"
	"github.com/diwise/context-broker/pkg/ngsild/client"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type CityWorkSvc interface {
	Start(ctx context.Context) error
	//TODO: This is not supposed to be a public interface (only exposed for testing it seems)
	getAndPublishCityWork(ctx context.Context) error
}

func NewCityWorkService(ctx context.Context, s SdlClient, timeInterval int, c client.ContextBrokerClient) CityWorkSvc {
	return &cw{
		log:           logging.GetFromContext(ctx),
		sdlClient:     s,
		timeInterval:  timeInterval,
		contextbroker: c,
	}
}

type cw struct {
	log           *slog.Logger
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
			log := logging.GetFromContext(ctx)
			log.Error("failed to get city work", slog.Int("retry", retryIntervalMinutes*60), "err", err.Error())
			sleepDuration = time.Duration(retryIntervalMinutes) * time.Minute
		}

		time.Sleep(sleepDuration)
	}
}

func (cw *cw) getAndPublishCityWork(ctx context.Context) error {
	response, err := cw.sdlClient.Get(ctx)
	logger := logging.GetFromContext(ctx)
	if err != nil {
		logger.Error("failed to get city work", "err", err.Error())
		return fmt.Errorf("failed to get city work")
	}

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	for _, f := range response.Features {
		featureID := f.ID()
		if _, exists := previous[featureID]; exists {
			continue
		}

		entityID := fiware.CityWorkIDPrefix + f.ID()

		attributes := toCityWorkModel(f)

		fragment, _ := entities.NewFragment(attributes...)

		_, err = cw.contextbroker.MergeEntity(ctx, entityID, fragment, headers)
		if err != nil {
			if !errors.Is(err, ngsierrors.ErrNotFound) {
				logger.Error("failed to merge entity", "entityID", entityID, "err", err.Error())
				logger.Info("waiting for context broker to recover...")
				time.Sleep(10 * time.Second)
				continue
			}
			entity, err := entities.New(entityID, fiware.CityWorkTypeName, attributes...)
			if err != nil {
				logger.Error("entities.New failed", "entityID", entityID, "err", err.Error())
				continue
			}

			_, err = cw.contextbroker.CreateEntity(ctx, entity, headers)
			if err != nil {
				logger.Error("failed to post city work to context broker", "entityID", entityID, "err", err.Error())
				continue
			}
		}

		previous[featureID] = featureID
	}

	return nil
}

func toCityWorkModel(sf sdlFeature) []entities.EntityDecoratorFunc {
	long, lat, _ := sf.Geometry.AsPoint()

	startDate := strings.ReplaceAll(sf.Properties.Start, "Z", "") + "T00:00:00Z"
	endDate := strings.ReplaceAll(sf.Properties.End, "Z", "") + "T23:59:59Z"

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 5),
		decorators.Location(lat, long),
		decorators.Description(sf.Properties.Description),
		decorators.DateTime("startDate", startDate),
		decorators.DateTime("endDate", endDate),
		decorators.DateTime("dateCreated", time.Now().UTC().Format(time.RFC3339)),
	)

	return attributes
}
