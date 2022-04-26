package citywork

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/diwise/integration-cip-sdl/internal/pkg/fiware"
	geojson "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/geojson"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"

	"github.com/rs/zerolog"
)

type CityWorkSvc interface {
	Start(ctx context.Context) error
	publishCityWorkToContextBroker(ctx context.Context, citywork fiware.CityWork) error
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

		response, err := cw.sdlClient.Get(ctx)
		if err != nil {
			cw.log.Error().Err(err).Msg("failed to get city work")
			continue
		}

		m, err := toModel(response)
		if err != nil {
			cw.log.Error().Err(err).Msg("failed to convert to model")
			continue
		}

		for _, f := range m.Features {
			featureID := f.ID()
			if _, exists := previous[featureID]; exists {
				continue
			}

			cwModel := toCityWorkModel(f)
			err = cw.publishCityWorkToContextBroker(ctx, cwModel)
			if err != nil {
				cw.log.Error().Err(err).Msg("failed to publish")
				continue
			}

			previous[featureID] = featureID
		}
	}
}

func toModel(resp []byte) (*sdlResponse, error) {
	var m sdlResponse

	err := json.Unmarshal(resp, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func toCityWorkModel(sf sdlFeature) fiware.CityWork {
	long, lat, _ := sf.Geometry.AsPoint()

	cw := fiware.NewCityWork(sf.ID())
	cw.StartDate = *ngsitypes.CreateDateTimeProperty(sf.Properties.Start + "T00:00:00Z")
	cw.EndDate = *ngsitypes.CreateDateTimeProperty(sf.Properties.End + "T23:59:59Z")
	cw.Location = geojson.CreateGeoJSONPropertyFromWGS84(long, lat)
	cw.DateCreated = *ngsitypes.CreateDateTimeProperty(time.Now().UTC().Format(time.RFC3339))
	cw.Description = ngsitypes.NewTextProperty(sf.Properties.Description)

	return cw
}

func (cw *cw) publishCityWorkToContextBroker(ctx context.Context, citywork fiware.CityWork) error {
	if err := cw.contextbroker.Post(ctx, citywork); err != nil {
		cw.log.Error().Err(err)
		return err
	}
	return nil
}
