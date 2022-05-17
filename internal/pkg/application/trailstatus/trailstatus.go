package trailstatus

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/integration-cip-sdl/internal/domain"
	"github.com/rs/zerolog"
)

type TrailPreparationService interface {
	Start(ctx context.Context, sourceBody []byte)
}

func NewTrailPreparationService(zlog zerolog.Logger, db Datastore) TrailPreparationService {
	ts := &trailServiceImpl{
		keepRunning: true,

		db:  db,
		log: zlog,
	}

	return ts
}

type trailServiceImpl struct {
	keepRunning bool

	db  Datastore
	log zerolog.Logger
}

func (ts *trailServiceImpl) Start(ctx context.Context, sourceBody []byte) {
	ts.updateTrailStatusFromSource(ctx, sourceBody)

	for ts.keepRunning {
		time.Sleep(60 * time.Second)
		ts.updateTrailStatusFromSource(ctx, sourceBody)
	}
}

func (ts *trailServiceImpl) updateTrailStatusFromSource(ctx context.Context, sourceBody []byte) error {

	status := struct {
		Ski map[string]struct {
			Active          bool   `json:"isActive"`
			ExternalID      string `json:"externalId"`
			LastPreparation string `json:"lastPreparation"`
		} `json:"Ski"`
	}{}

	_ = json.Unmarshal(sourceBody, &status)

	for k, v := range status.Ski {
		if v.ExternalID != "" {
			trailID := domain.SundsvallAnlaggningPrefix + v.ExternalID

			ts.db.SetTrailOpenStatus(trailID, v.Active)

			if v.Active {
				lastPrepared, err := time.Parse(time.RFC3339, v.LastPreparation)
				if err != nil {
					ts.log.Warn().Err(err).Msgf("failed to parse trail preparation timestamp for %s", k)
					continue
				}

				_, err = ts.db.UpdateTrailLastPreparationTime(trailID, lastPrepared)
				if err != nil {
					ts.log.Error().Err(err).Msgf("failed to update trail status for %s", k)
					continue
				}
			}
		}
	}

	return nil
}
