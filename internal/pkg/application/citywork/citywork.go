package citywork

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/diwise/integration-cip-sdl/internal/domain"	
	geojson "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/geojson"
	types "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
	fiware "github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/rs/zerolog"
)

type CityWorkSvc interface {
	Start(ctx context.Context) error
	getAndPublishCityWork(ctx context.Context) error
}

func NewCityWorkService(log zerolog.Logger, s SdlClient, c domain.ContextBrokerClient) CityWorkSvc {
	return &cw{
		log:           log,
		sdlClient:     s,
		contextbroker: c,
	}
}

type cw struct {
	log           zerolog.Logger
	sdlClient     SdlClient
	contextbroker domain.ContextBrokerClient
}

var previous map[string]string = make(map[string]string)

func (cw *cw) Start(ctx context.Context) error {
	for {
		time.Sleep(10 * time.Second)

		err := cw.getAndPublishCityWork(ctx)
		if err != nil {
			cw.log.Error().Err(err).Msg("failed to get city work, attempting again in 10 seconds")
			continue
		}
	}
}

func (cw *cw) getAndPublishCityWork(ctx context.Context) error {
	response, err := cw.sdlClient.Get(ctx)
	if err != nil {
		cw.log.Error().Err(err).Msg("failed to get city work")
		return fmt.Errorf("failed to get city work")
	}

	for _, f := range response.Features {
		featureID := f.ID()
		if _, exists := previous[featureID]; exists {
			continue
		}

		cwModel := toCityWorkModel(f)

		err = cw.contextbroker.AddEntity(ctx, cwModel)
		if err != nil {
			cw.log.Error().Err(err).Msg("failed to add entity")
			continue
		}

		previous[featureID] = featureID
	}

	return nil
}

func toCityWorkModel(sf sdlFeature) fiware.CityWork {
	long, lat, _ := sf.Geometry.AsPoint()

	startDate := strings.ReplaceAll(sf.Properties.Start, "Z", "") + "T00:00:00Z"
	endDate := strings.ReplaceAll(sf.Properties.End, "Z", "") + "T23:59:59Z"

	cw := fiware.NewCityWork(sf.ID())
	cw.StartDate = *types.CreateDateTimeProperty(startDate)
	cw.EndDate = *types.CreateDateTimeProperty(endDate)
	cw.Location = geojson.CreateGeoJSONPropertyFromWGS84(long, lat)
	cw.DateCreated = *types.CreateDateTimeProperty(time.Now().UTC().Format(time.RFC3339))
	cw.Description = types.NewTextProperty(sf.Properties.Description)

	return cw
}
