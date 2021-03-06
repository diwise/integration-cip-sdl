package facilities

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/diwise/context-broker/pkg/datamodels/diwise"
	"github.com/diwise/context-broker/pkg/ngsild/client"
	ngsitypes "github.com/diwise/context-broker/pkg/ngsild/types"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	. "github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/integration-cip-sdl/internal/domain"
)

func StoreTrailsFromSource(logger zerolog.Logger, ctxBrokerClient client.ContextBrokerClient, ctx context.Context, sourceURL string, featureCollection domain.FeatureCollection) error {

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	for _, feature := range featureCollection.Features {
		if feature.Properties.Published {
			if feature.Properties.Type == "Motionsspår" || feature.Properties.Type == "Skidspår" || feature.Properties.Type == "Långfärdsskridskoled" {
				exerciseTrail, err := parsePublishedExerciseTrail(logger, feature)
				if err != nil {
					logger.Error().Err(err).Msg("failed to parse motionsspår")
					continue
				}

				exerciseTrail.Source = fmt.Sprintf("%s/get/%d", sourceURL, feature.ID)

				fiwareTrail := convertDBTrailToFiwareExerciseTrail(*exerciseTrail)

				_, err = ctxBrokerClient.CreateEntity(ctx, fiwareTrail, headers)
				if err != nil {
					logger.Error().Err(err).Msg("failed to post exercise trail to context broker")
					continue
				}
			}
		}
	}

	return nil
}

func parsePublishedExerciseTrail(log zerolog.Logger, feature domain.Feature) (*domain.ExerciseTrail, error) {
	log.Info().Msgf("found published exercise trail %d %s", feature.ID, feature.Properties.Name)

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
			openStatus := map[string]string{"Ja": "open", "Nej": "closed"}
			trail.Status = openStatus[string(field.Value[1:len(field.Value)-1])]
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

func convertDBTrailToFiwareExerciseTrail(trail domain.ExerciseTrail) ngsitypes.Entity {

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 8),
		LocationLS(trail.Geometry.Lines), Description(trail.Description),
		DateTimeIfNotZero(properties.DateCreated, trail.DateCreated),
		DateTimeIfNotZero(properties.DateModified, trail.DateModified),
		DateTimeIfNotZero("dateLastPreparation", trail.DateLastPrepared),
	)

	if trail.AreaServed != "" {
		attributes = append(attributes, Text("areaServed", trail.AreaServed))
	}

	if len(trail.Category) > 0 {
		attributes = append(attributes, TextList("category", trail.Category))
	}

	if trail.Source != "" {
		attributes = append(attributes, Source(trail.Source))
	}

	if trail.Status != "" {
		attributes = append(attributes, Status(trail.Status))
	}

	et, _ := diwise.NewExerciseTrail(trail.ID, trail.Name, trail.Length, trail.Description, attributes...)
	return et
}
