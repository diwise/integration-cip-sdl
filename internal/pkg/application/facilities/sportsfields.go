package facilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/diwise/context-broker/pkg/datamodels/diwise"
	"github.com/diwise/context-broker/pkg/ngsild/client"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	. "github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/context-broker/pkg/ngsild/types/relationships"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

var ErrSportsFieldIsOfIgnoredType error = errors.New("sportsfield is of non supported type")

func (s *storageImpl) StoreSportsFieldsFromSource(ctx context.Context, ctxBrokerClient client.ContextBrokerClient, sourceURL string, featureCollection domain.FeatureCollection) error {

	logger := logging.GetFromContext(ctx)

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	for _, feature := range featureCollection.Features {
		if feature.Properties.Type == "Aktivitetsyta" {
			sportsField, err := parseSportsField(ctx, feature)
			if err != nil {
				if !errors.Is(err, ErrSportsFieldIsOfIgnoredType) {
					logger.Error("failed to parse aktivitetsyta", "err", err.Error())
				}
				continue
			}

			entityID := diwise.SportsFieldIDPrefix + sportsField.ID

			if okToDel, alreadyDeleted := s.shouldBeDeleted(feature); okToDel {
				if !alreadyDeleted {
					_, err := ctxBrokerClient.DeleteEntity(ctx, entityID)
					if err != nil {
						logger.Info("could not delete entity", "entityID", entityID)
					}
				}
				continue
			}

			sportsField.Source = fmt.Sprintf("%s/get/%d", sourceURL, feature.ID)

			attributes := convertDBSportsFieldToFiwareSportsField(*sportsField)

			fragment, _ := entities.NewFragment(attributes...)

			_, err = ctxBrokerClient.MergeEntity(ctx, entityID, fragment, headers)

			// Throttle so we dont kill the broker
			time.Sleep(500 * time.Millisecond)

			if err != nil {
				if !errors.Is(err, ngsierrors.ErrNotFound) {
					logger.Error("failed to merge entity", "entityID", entityID, "err", err.Error())
					continue
				}
				entity, err := entities.New(entityID, diwise.SportsFieldTypeName, attributes...)
				if err != nil {
					logger.Error("entities.New failed", "entityID", entityID, "err", err.Error())
					continue
				}

				_, err = ctxBrokerClient.CreateEntity(ctx, entity, headers)
				if err != nil {
					logger.Error("failed to post sports field to context broker", "entityID", entityID, "err", err.Error())
					continue
				}
			}

		}
	}

	return nil
}

func parseSportsField(ctx context.Context, feature domain.Feature) (*domain.SportsField, error) {
	logger := logging.GetFromContext(ctx)
	logger.Info("found published sports field", slog.Int64("featureID", feature.ID), "name", feature.Properties.Name)

	sportsField := &domain.SportsField{
		ID:          fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID),
		Name:        feature.Properties.Name,
		Description: "",
	}

	if !feature.Properties.Published {
		return sportsField, nil
	}

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

	if feature.Properties.Manager != nil {
		sportsField.ManagedBy = fmt.Sprintf("urn:ngsi-ld:Organisation:se:sundsvall:facilities:org:%d", feature.Properties.Manager.OrganisationID)
	}

	if feature.Properties.Owner != nil {
		sportsField.Owner = fmt.Sprintf("urn:ngsi-ld:Organisation:se:sundsvall:facilities:org:%d", feature.Properties.Owner.OrganisationID)
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
		} else if field.ID == 153 {
			publicAccess := map[string]string{
				"Hela dygnet":          "always",
				"Nej":                  "no",
				"Särskilda öppettider": "opening-hours",
				"Utanför skoltid":      "after-school",
			}
			paValue := string(field.Value[1 : len(field.Value)-1])

			var ok bool
			sportsField.PublicAccess, ok = publicAccess[paValue]
			if !ok {
				return nil, fmt.Errorf("unknown public access value: %s", paValue)
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

	if field.ManagedBy != "" {
		attributes = append(attributes, entities.R("managedBy", relationships.NewSingleObjectRelationship(field.ManagedBy)))
	}

	if field.Owner != "" {
		attributes = append(attributes, entities.R("owner", relationships.NewSingleObjectRelationship(field.Owner)))
	}

	if len(field.Category) > 0 {
		attributes = append(attributes, TextList("category", field.Category))
	}

	if len(field.PublicAccess) > 0 {
		attributes = append(attributes, Text("publicAccess", field.PublicAccess))
	}

	if field.Source != "" {
		attributes = append(attributes, Source(field.Source))
	}

	return attributes
}
