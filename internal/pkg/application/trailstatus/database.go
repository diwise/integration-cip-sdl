package trailstatus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/diwise/integration-cip-sdl/internal/domain"
)

//Datastore is an interface that abstracts away the database implementation
type Datastore interface {
	SetTrailOpenStatus(trailID string, isOpen bool) error
	UpdateTrailLastPreparationTime(trailID string, dateLastPreparation time.Time) error
}

//NewDatabaseConnection does not open a new connection ...
func NewDatabaseConnection(sourceURL, apiKey string, logger zerolog.Logger, ctxClient domain.ContextBrokerClient, ctx context.Context) (Datastore, error) {
	if sourceURL == "" || apiKey == "" {
		return nil, fmt.Errorf("all environment variables must be set")
	}

	logger.Info().Msgf("loading data from %s ...", sourceURL)

	req, err := http.NewRequest("GET", sourceURL+"/list", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("loading data from %s failed with status %d", sourceURL, resp.StatusCode)
	}

	featureCollection := &domain.FeatureCollection{}
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, featureCollection)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response from %s. (%s)", sourceURL, err.Error())
	}

	db := &myDB{}

	for _, feature := range featureCollection.Features {
		if feature.Properties.Published {
			if feature.Properties.Type == "Motionsspår" || feature.Properties.Type == "Skidspår" || feature.Properties.Type == "Långfärdsskridskoled" {
				exerciseTrail, err := parsePublishedExerciseTrail(logger, feature)
				if err != nil {
					logger.Error().Err(err).Msg("failed to parse motionsspår")
					continue
				}

				exerciseTrail.Source = fmt.Sprintf("%s/get/%d", sourceURL, feature.ID)

				db.trails = append(db.trails, *exerciseTrail)
			}
		}
	}

	return db, nil
}

func parsePublishedExerciseTrail(log zerolog.Logger, feature domain.Feature) (*domain.ExerciseTrail, error) {
	log.Info().Msgf("found published exercise trail %d %s\n", feature.ID, feature.Properties.Name)

	trail := &domain.ExerciseTrail{
		ID:          fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID),
		Name:        feature.Properties.Name,
		Description: "",
	}

	var timeFormat string = "2006-01-02 15:04:05"

	if feature.Properties.Created != nil {
		created, err := time.Parse(timeFormat, *feature.Properties.Created)
		if err == nil {
			trail.DateCreated = created.UTC()
		}
	}

	if feature.Properties.Updated != nil {
		modified, err := time.Parse(timeFormat, *feature.Properties.Updated)
		if err == nil {
			trail.DateModified = modified.UTC()
		}
	}

	err := json.Unmarshal(feature.Geometry.Coordinates, &trail.Geometry.Lines)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal geometry %s: %s", string(feature.Geometry.Coordinates), err.Error())
	}

	fields := []domain.FeaturePropField{}
	err = json.Unmarshal(feature.Properties.Fields, &fields)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal property fields %s: %s", string(feature.Properties.Fields), err.Error())
	}

	categories := []string{}

	if feature.Properties.Type == "Långfärdsskridskoled" {
		categories = append(categories, "ice-skating")
	}

	for _, field := range fields {
		if field.ID == 99 {
			length, _ := strconv.ParseInt(string(field.Value[0:len(field.Value)]), 10, 64)
			trail.Length = float64(length) / 1000.0
		} else if field.ID == 102 {
			isOpen := string(field.Value[1 : len(field.Value)-1])
			openStatus := map[string]string{"Ja": "open", "Nej": "closed"}
			trail.Status = openStatus[isOpen]
		} else if field.ID == 103 {
			if propertyValueMatches(field, "Ja") {
				categories = append(categories, "floodlit")
			}
		} else if field.ID == 110 {
			trail.Description = string(field.Value[1 : len(field.Value)-1])
		} else if field.ID == 134 {
			trail.AreaServed = string(field.Value[1 : len(field.Value)-1])
		} else if field.ID == 248 || field.ID == 250 {
			if propertyValueMatches(field, "Ja") {
				categories = append(categories, "ski-classic")
			}
		} else if field.ID == 249 || field.ID == 251 {
			if propertyValueMatches(field, "Ja") {
				categories = append(categories, "ski-skate")
			}
		}
	}

	if len(categories) > 0 {
		trail.Category = categories
	}

	return trail, nil
}

func propertyValueMatches(field domain.FeaturePropField, expectation string) bool {
	value := string(field.Value[0:len(field.Value)])
	return value == expectation || value == ("\""+expectation+"\"")
}

type myDB struct {
	trails    []domain.ExerciseTrail
	ctxClient domain.ContextBrokerClient
	ctx       context.Context
}

func (db *myDB) SetTrailOpenStatus(trailID string, isOpen bool) error {
	for idx, trail := range db.trails {
		if strings.Compare(trail.ID, trailID) == 0 {
			status := "closed"
			if isOpen {
				status = "open"
			}

			db.trails[idx].Status = status
			db.ctxClient.AddEntity(db.ctx, db.trails[idx])

			return nil
		}
	}

	return errors.New("not found")
}

func (db *myDB) UpdateTrailLastPreparationTime(trailID string, dateLastPreparation time.Time) error {
	for idx, trail := range db.trails {
		if strings.Compare(trail.ID, trailID) == 0 {
			if trail.DateLastPrepared.After(dateLastPreparation) {
				return fmt.Errorf("last preparation date may not move backwards")
			}

			db.trails[idx].DateLastPrepared = dateLastPreparation

			return nil
		}
	}

	return errors.New("not found")
}
