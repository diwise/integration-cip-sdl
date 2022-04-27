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

		var m sdlResponse
		err = json.Unmarshal(response, &m)
		if err != nil {
			cw.log.Error().Err(err).Msg("failed to unmarshal model")
			continue
		}

		for _, f := range m.Features {
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
	}
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
