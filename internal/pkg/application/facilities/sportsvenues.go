package facilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/diwise/context-broker/pkg/datamodels/diwise"
	"github.com/diwise/context-broker/pkg/ngsild/client"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/context-broker/pkg/ngsild/types/relationships"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"

	//lint:ignore ST1001 it is OK when we do it
	. "github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
)

var ErrSportsVenueIsOfIgnoredType error = errors.New("sports venue is of non supported type")

func (s *storageImpl) StoreSportsVenuesFromSource(ctx context.Context, ctxBrokerClient client.ContextBrokerClient, sourceURL string, featureCollection domain.FeatureCollection) error {

	logger := logging.GetFromContext(ctx)

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	isSupportedType := func(t string) bool {
		return t == "Badhus" || t == "Ishall" || t == "Sporthall"
	}

	for _, feature := range featureCollection.Features {
		if isSupportedType(feature.Properties.Type) {
			sportsVenue, err := parseSportsVenue(ctx, feature)
			if err != nil {
				if !errors.Is(err, ErrSportsVenueIsOfIgnoredType) {
					logger.Error("failed to parse sports venue", slog.Int64("featureID", feature.ID), "err", err.Error())
				}
				continue
			}

			entityID := diwise.SportsVenueIDPrefix + sportsVenue.ID

			if okToDel, alreadyDeleted := s.shouldBeDeleted(ctx, feature); okToDel {
				if !alreadyDeleted {
					_, err := ctxBrokerClient.DeleteEntity(ctx, entityID)
					if err != nil {
						logger.Info("could not delete entity", "entityID", entityID, "err", err.Error())
					}
				}
				continue
			}

			sportsVenue.Source = fmt.Sprintf("%s/get/%d", sourceURL, feature.ID)

			attributes := convertDBSportsVenueToFiwareSportsVenue(*sportsVenue)

			fragment, _ := entities.NewFragment(attributes...)

			_, err = ctxBrokerClient.MergeEntity(ctx, entityID, fragment, headers)

			// Throttle so we dont kill the broker
			time.Sleep(100 * time.Millisecond)

			if err != nil {
				if !errors.Is(err, ngsierrors.ErrNotFound) {
					logger.Error("failed to merge entity", "entityID", entityID, "err", err.Error())
					logger.Info("waiting for context broker to recover...")
					time.Sleep(10 * time.Second)
					continue
				}
				entity, err := entities.New(entityID, diwise.SportsVenueTypeName, attributes...)
				if err != nil {
					logger.Error("entities.New failed", "entityID", entityID, "err", err.Error())
					continue
				}

				_, err = ctxBrokerClient.CreateEntity(ctx, entity, headers)
				if err != nil {
					logger.Error("failed to post sports venue to context broker", "entityID", entityID, "err", err.Error())
					continue
				}
			}
		}
	}

	return nil
}

func parseSportsVenue(ctx context.Context, feature domain.Feature) (*domain.SportsVenue, error) {
	logger := logging.GetFromContext(ctx)
	logger.Info("found published sports venue", slog.Int64("featureID", feature.ID), "name", feature.Properties.Name)

	sportsVenue := &domain.SportsVenue{
		ID:          fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID),
		Name:        feature.Properties.Name,
		Description: "",
	}

	if !feature.Properties.Published {
		return sportsVenue, nil
	}

	if feature.Properties.Created != nil {
		created, err := time.Parse(timeFormat, *feature.Properties.Created)
		if err == nil {
			sportsVenue.DateCreated = created.UTC()
		}
	}

	if feature.Properties.Updated != nil {
		modified, err := time.Parse(timeFormat, *feature.Properties.Updated)
		if err == nil {
			sportsVenue.DateModified = modified.UTC()
		}
	}

	err := json.Unmarshal(feature.Geometry.Coordinates, &sportsVenue.Geometry.Lines)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal geometry %s: %s", string(feature.Geometry.Coordinates), err.Error())
	}

	if feature.Properties.Manager != nil {
		sportsVenue.ManagedBy = fmt.Sprintf("urn:ngsi-ld:Organisation:se:sundsvall:facilities:org:%d", feature.Properties.Manager.OrganisationID)
	}

	if feature.Properties.Owner != nil {
		sportsVenue.Owner = fmt.Sprintf("urn:ngsi-ld:Organisation:se:sundsvall:facilities:org:%d", feature.Properties.Owner.OrganisationID)
	}

	fields := []domain.FeaturePropField{}
	err = json.Unmarshal(feature.Properties.Fields, &fields)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal property fields %s: %s", string(feature.Properties.Fields), err.Error())
	}

	sportsVenue.SeeAlso = []string{}

	for _, field := range fields {
		if field.ID == 78 {
			sportsVenue.Description = string(field.Value[1 : len(field.Value)-1])
		} else if field.ID == 151 {
			url := string(field.Value[1 : len(field.Value)-1])
			url = strings.ReplaceAll(url, "\\/", "/")
			sportsVenue.SeeAlso = []string{url}
		} else if field.ID == 200 {
			publicAccess := map[string]string{
				"Hela dygnet":          "always",
				"Nej":                  "no",
				"Särskilda öppettider": "opening-hours",
				"Utanför skoltid":      "after-school",
			}
			paValue := string(field.Value[1 : len(field.Value)-1])

			var ok bool
			sportsVenue.PublicAccess, ok = publicAccess[paValue]
			if !ok {
				return nil, fmt.Errorf("unknown public access value: %s", paValue)
			}
		}
	}

	supportedCategories := map[string][]string{
		"Badhus":    {"swimming-pool"},
		"Ishall":    {"ice-rink"},
		"Sporthall": {"sports-hall"},
	}

	categories, ok := supportedCategories[feature.Properties.Type]
	if ok {
		sportsVenue.Category = categories
	}

	return sportsVenue, nil
}

func convertDBSportsVenueToFiwareSportsVenue(venue domain.SportsVenue) []entities.EntityDecoratorFunc {

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 8),
		Name(venue.Name), Description(venue.Description),
		LocationMP(venue.Geometry.Lines),
		DateTimeIfNotZero(properties.DateCreated, venue.DateCreated),
		DateTimeIfNotZero(properties.DateModified, venue.DateModified),
	)

	if len(venue.Category) > 0 {
		attributes = append(attributes, TextList("category", venue.Category))
	}

	if venue.ManagedBy != "" {
		attributes = append(attributes, entities.R("managedBy", relationships.NewSingleObjectRelationship(venue.ManagedBy)))
	}

	if venue.Owner != "" {
		attributes = append(attributes, entities.R("owner", relationships.NewSingleObjectRelationship(venue.Owner)))
	}

	if len(venue.PublicAccess) > 0 {
		attributes = append(attributes, Text("publicAccess", venue.PublicAccess))
	}

	if len(venue.SeeAlso) > 0 {
		attributes = append(attributes, TextList("seeAlso", venue.SeeAlso))
	}

	if venue.Source != "" {
		attributes = append(attributes, Source(venue.Source))
	}

	return attributes
}
