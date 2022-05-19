package trailstatus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/diwise/integration-cip-sdl/internal/domain"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/diwise"
	"github.com/rs/zerolog"
)

type TrailPreparationService interface {
	UpdateTrailStatusFromSource(ctx context.Context) error
}

func NewTrailPreparationService(log zerolog.Logger, db Datastore, cs domain.ContextBrokerClient, url string) TrailPreparationService {
	ts := &trailServiceImpl{
		url: url,
		cs:  cs,
		db:  db,
		log: log,
	}

	return ts
}

type trailServiceImpl struct {
	url string
	cs  domain.ContextBrokerClient
	db  Datastore
	log zerolog.Logger
}

func (ts *trailServiceImpl) UpdateTrailStatusFromSource(ctx context.Context) error {
	req, err := http.NewRequest("GET", ts.url, nil)
	if err != nil {
		ts.log.Error().Err(err).Msg("failed to create http request")
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.log.Error().Err(err).Msg("failed to request trail status update")
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ts.log.Error().Msgf("loading data from %s failed with status %d", ts.url, resp.StatusCode)
		return err
	}

	status := struct {
		Ski map[string]struct {
			Active          bool   `json:"isActive"`
			ExternalID      string `json:"externalId"`
			LastPreparation string `json:"lastPreparation"`
		} `json:"Ski"`
	}{}

	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &status)
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
