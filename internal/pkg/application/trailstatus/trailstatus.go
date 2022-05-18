package trailstatus

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/integration-cip-sdl/internal/domain"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/diwise"
	"github.com/rs/zerolog"
)

type TrailPreparationService interface {
	UpdateTrailStatusFromSource(ctx context.Context, sourceBody []byte) error
}

func NewTrailPreparationService(log zerolog.Logger, db Datastore, cs domain.ContextBrokerClient) TrailPreparationService {
	ts := &trailServiceImpl{
		keepRunning: true,

		cs:  cs,
		db:  db,
		log: log,
	}

	return ts
}

type trailServiceImpl struct {
	keepRunning bool

	cs  domain.ContextBrokerClient
	db  Datastore
	log zerolog.Logger
}

func (ts *trailServiceImpl) UpdateTrailStatusFromSource(ctx context.Context, sourceBody []byte) error {
	status := struct {
		Ski map[string]struct {
			Active          bool   `json:"isActive"`
			ExternalID      string `json:"externalId"`
			LastPreparation string `json:"lastPreparation"`
		} `json:"Ski"`
	}{}

	err := json.Unmarshal(sourceBody, &status)
	if err != nil {
		return err
	}

	for k, v := range status.Ski {
		if v.ExternalID != "" {
			trail := &diwise.ExerciseTrail{}
			trailID := domain.SundsvallAnlaggningPrefix + v.ExternalID

			ts.db.SetTrailOpenStatus(trailID, v.Active)

			if v.Active {
				lastPrepared, err := time.Parse(time.RFC3339, v.LastPreparation)
				if err != nil {
					ts.log.Warn().Err(err).Msgf("failed to parse trail preparation timestamp for %s", k)
					continue
				}

				trail, err = ts.db.UpdateTrailLastPreparationTime(trailID, lastPrepared)
				if err != nil {
					ts.log.Error().Err(err).Msgf("failed to update trail status for %s", k)
					continue
				}
			}
			ts.cs.UpdateEntity(ctx, trail, trail.ID)
		}
	}

	return err
}
