package trailstatus

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/diwise/integration-cip-sdl/internal/pkg/infrastructure/repositories/database"
	"github.com/rs/zerolog"
)

type TrailPreparationService interface {
	Shutdown()
}

func NewTrailPreparationService(zlog zerolog.Logger, url string, db database.Datastore) TrailPreparationService {
	ts := &trailServiceImpl{
		keepRunning: true,
		url:         url,
		db:          db,
		log:         zlog,
	}

	go ts.run()

	return ts
}

type trailServiceImpl struct {
	keepRunning bool
	url         string
	db          database.Datastore
	log         zerolog.Logger
}

func (ts *trailServiceImpl) run() {
	ts.updateTrailStatusFromSource()

	for ts.keepRunning {
		time.Sleep(60 * time.Second)
		ts.updateTrailStatusFromSource()
	}
}

func (ts *trailServiceImpl) updateTrailStatusFromSource() {
	req, err := http.NewRequest("GET", ts.url, nil)
	if err != nil {
		ts.log.Error().Err(err).Msg("failed to create http request")
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.log.Error().Err(err).Msg("failed to request trail status update")
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ts.log.Error().Msgf("loading data from %s failed with status %d", ts.url, resp.StatusCode)
		return
	}

	status := struct {
		Ski map[string]struct {
			Active          bool   `json:"isActive"`
			ExternalID      string `json:"externalId"`
			LastPreparation string `json:"lastPreparation"`
		} `json:"Ski"`
	}{}

	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &status)

	for k, v := range status.Ski {
		if v.ExternalID != "" {
			trailID := database.SundsvallAnlaggningPrefix + v.ExternalID

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
				}
			}
		}
	}
}

func (ts *trailServiceImpl) Shutdown() {
	ts.keepRunning = false
}
