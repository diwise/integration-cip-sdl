package trailstatus

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/integration-cip-sdl/internal/domain"
	"github.com/rs/zerolog"
)

type TrailPreparationService interface {
	Start(ctx context.Context)
	Shutdown()
}

func NewTrailPreparationService(zlog zerolog.Logger, sc sdlClient, db Datastore) TrailPreparationService {
	ts := &trailServiceImpl{
		keepRunning: true,
		sc:          sc,
		db:          db,
		log:         zlog,
	}

	return ts
}

type trailServiceImpl struct {
	keepRunning bool
	sc          sdlClient
	db          Datastore
	log         zerolog.Logger
}

func (ts *trailServiceImpl) Start(ctx context.Context) {
	ts.updateTrailStatusFromSource(ctx)

	for ts.keepRunning {
		time.Sleep(60 * time.Second)
		ts.updateTrailStatusFromSource(ctx)
	}
}

func (ts *trailServiceImpl) updateTrailStatusFromSource(ctx context.Context) error {

	status := struct {
		Ski map[string]struct {
			Active          bool   `json:"isActive"`
			ExternalID      string `json:"externalId"`
			LastPreparation string `json:"lastPreparation"`
		} `json:"Ski"`
	}{}

	body, err := ts.sc.Get(ctx)
	if err != nil {
		return err
	}

	_ = json.Unmarshal(body, &status)

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

				err = ts.db.UpdateTrailLastPreparationTime(trailID, lastPrepared)
				if err != nil {
					ts.log.Error().Err(err).Msgf("failed to update trail status for %s", k)
					continue
				}
			}
		}
	}

	return nil
}

func (ts *trailServiceImpl) Shutdown() {
	ts.keepRunning = false
}
