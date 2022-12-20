package facilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/diwise/context-broker/pkg/datamodels/diwise"
	"github.com/diwise/context-broker/pkg/ngsild/client"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	. "github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/context-broker/pkg/ngsild/types/relationships"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
)

var ErrSportsVenueIsOfIgnoredType error = errors.New("sports venue is of non supported type")

func StoreSportsVenuesFromSource(logger zerolog.Logger, ctxBrokerClient client.ContextBrokerClient, ctx context.Context, sourceURL string, featureCollection domain.FeatureCollection) error {

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	isSupportedType := func(t string) bool {
		return t == "Badhus" || t == "Ishall" || t == "Sporthall"
	}

	for _, feature := range featureCollection.Features {
		if isSupportedType(feature.Properties.Type) {
			sportsVenue, published, err := parseSportsVenue(logger, feature)
			if err != nil {
				if !errors.Is(err, ErrSportsVenueIsOfIgnoredType) {
					logger.Error().Err(err).Msg("failed to parse feature")
				}
				continue
			}

			entityID := diwise.SportsVenueIDPrefix + sportsVenue.ID

			if !published {
				if shouldBeDeleted(feature) {
					_, err := ctxBrokerClient.DeleteEntity(ctx, entityID)
					if err != nil {
						logger.Info().Msgf("could not delete entity %s", entityID)
					}
				}
				continue
			}

			sportsVenue.Source = fmt.Sprintf("%s/get/%d", sourceURL, feature.ID)

			attributes := convertDBSportsVenueToFiwareSportsVenue(*sportsVenue)

			fragment, _ := entities.NewFragment(attributes...)

			_, err = ctxBrokerClient.MergeEntity(ctx, entityID, fragment, headers)

			// Throttle so we dont kill the broker
			time.Sleep(500 * time.Millisecond)

			if err != nil {
				if !errors.Is(err, ngsierrors.ErrNotFound) {
					logger.Error().Err(err).Msg("failed to merge entity")
					continue
				}
				entity, err := entities.New(entityID, diwise.SportsVenueTypeName, attributes...)
				if err != nil {
					logger.Error().Err(err).Msg("entities.New failed")
					continue
				}

				_, err = ctxBrokerClient.CreateEntity(ctx, entity, headers)
				if err != nil {
					logger.Error().Err(err).Msg("failed to post sports venue to context broker")
					continue
				}
			}
		}
	}

	return nil
}

func parseSportsVenue(log zerolog.Logger, feature domain.Feature) (*domain.SportsVenue, bool, error) {
	if feature.Properties.Published {
		sv, err := parsePublishedSportsVenue(log, feature)
		return sv, true, err
	}
	sportsVenue := &domain.SportsVenue{
		ID:          fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID),
		Name:        feature.Properties.Name,
		Description: "",
	}
	return sportsVenue, false, nil
}

func parsePublishedSportsVenue(log zerolog.Logger, feature domain.Feature) (*domain.SportsVenue, error) {
	log.Info().Msgf("found published sports venue %d %s", feature.ID, feature.Properties.Name)

	sportsVenue := &domain.SportsVenue{
		ID:          fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID),
		Name:        feature.Properties.Name,
		Description: "",
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
