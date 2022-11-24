package facilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/diwise/context-broker/pkg/datamodels/diwise"
	"github.com/diwise/context-broker/pkg/ngsild/client"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	. "github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
)

var ErrSportsFieldIsOfIgnoredType error = errors.New("sportsfield is of non supported type")

func StoreSportsFieldsFromSource(logger zerolog.Logger, ctxBrokerClient client.ContextBrokerClient, ctx context.Context, sourceURL string, featureCollection domain.FeatureCollection) error {

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	for _, feature := range featureCollection.Features {
		if feature.Properties.Published {
			if feature.Properties.Type == "Aktivitetsyta" {
				sportsField, err := parsePublishedSportsField(logger, feature)
				if err != nil {
					if !errors.Is(err, ErrSportsFieldIsOfIgnoredType) {
						logger.Error().Err(err).Msg("failed to parse aktivitetsyta")
					}
					continue
				}

				sportsField.Source = fmt.Sprintf("%s/get/%d", sourceURL, feature.ID)

				attributes := convertDBSportsFieldToFiwareSportsField(*sportsField)

				fragment, _ := entities.NewFragment(attributes...)

				entityID := diwise.SportsFieldIDPrefix + sportsField.ID

				_, err = ctxBrokerClient.MergeEntity(ctx, entityID, fragment, headers)
				if err != nil {
					if !errors.Is(err, ngsierrors.ErrNotFound) {
						logger.Error().Err(err).Msg("failed to merge entity")
						continue
					}
					entity, err := entities.New(entityID, diwise.SportsFieldTypeName, attributes...)
					if err != nil {
						logger.Error().Err(err).Msg("entities.New failed")
						continue
					}

					_, err = ctxBrokerClient.CreateEntity(ctx, entity, headers)
					if err != nil {
						logger.Error().Err(err).Msg("failed to post sports field to context broker")
						continue
					}
				}
			}
		}
	}

	return nil
}

func parsePublishedSportsField(log zerolog.Logger, feature domain.Feature) (*domain.SportsField, error) {
	log.Info().Msgf("found published sports field %d %s", feature.ID, feature.Properties.Name)

	sportsField := &domain.SportsField{
		ID:          fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID),
		Name:        feature.Properties.Name,
		Description: "",
	}

	var timeFormat string = "2006-01-02 15:04:05"

	if feature.Properties.Created != nil {
		created, err := time.Parse(timeFormat, *feature.Properties.Created)
		if err == nil {
			sportsField.DateCreated = created.UTC()
		}
	}

	if feature.Properties.Updated != nil {
		modified, err := time.Parse(timeFormat, *feature.Properties.Updated)
		if err == nil {
			sportsField.DateModified = modified.UTC()
		}
	}

	err := json.Unmarshal(feature.Geometry.Coordinates, &sportsField.Geometry.Lines)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal geometry %s: %s", string(feature.Geometry.Coordinates), err.Error())
	}

	fields := []domain.FeaturePropField{}
	err = json.Unmarshal(feature.Properties.Fields, &fields)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal property fields %s: %s", string(feature.Properties.Fields), err.Error())
	}

	categories := []string{}
	var ignoreThisField bool = true
	var isIceRink bool = false

	for _, field := range fields {
		if field.ID == 279 {
			if propertyValueMatches(field, "Ja") {
				categories = append(categories, "floodlit")
			}
		} else if field.ID == 1 {
			sportsField.Description = string(field.Value[1 : len(field.Value)-1])
		} else if field.ID == 137 || field.ID == 138 || field.ID == 139 {
			if propertyValueMatches(field, "Ja") {
				isIceRink = true
				ignoreThisField = false

				if field.ID == 137 {
					categories = append(categories, "skating")
				} else if field.ID == 138 {
					categories = append(categories, "hockey")
				} else if field.ID == 139 {
					categories = append(categories, "bandy")
				}
			}
		}
	}

	if ignoreThisField {
		return nil, ErrSportsFieldIsOfIgnoredType
	}

	if isIceRink {
		categories = append(categories, "ice-rink")
	}

	if len(categories) > 0 {
		sportsField.Category = categories
	}

	return sportsField, nil
}

func convertDBSportsFieldToFiwareSportsField(field domain.SportsField) []entities.EntityDecoratorFunc {

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 7),
		LocationMP(field.Geometry.Lines), Description(field.Description),
		DateTimeIfNotZero(properties.DateCreated, field.DateCreated),
		DateTimeIfNotZero(properties.DateModified, field.DateModified),
		DateTimeIfNotZero("dateLastPreparation", field.DateLastPrepared),
		Name(field.Name),
		Description(field.Description),
	)

	if len(field.Category) > 0 {
		attributes = append(attributes, TextList("category", field.Category))
	}

	if field.Source != "" {
		attributes = append(attributes, Source(field.Source))
	}

	return attributes
}
