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
	"github.com/diwise/integration-cip-sdl/internal/domain"
)

var ErrSportsVenueIsOfIgnoredType error = errors.New("sportsvenue is of non supported type")

func StoreSportsVenuesFromSource(logger zerolog.Logger, ctxBrokerClient client.ContextBrokerClient, ctx context.Context, sourceURL string, featureCollection domain.FeatureCollection) error {

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	isSupportedType := func(t string) bool {
		return t == "Badhus" || t == "Ishall" || t == "Sporthall"
	}

	for _, feature := range featureCollection.Features {
		if feature.Properties.Published {
			if isSupportedType(feature.Properties.Type) {
				sportsVenue, err := parsePublishedSportsVenue(logger, feature)
				if err != nil {
					if !errors.Is(err, ErrSportsVenueIsOfIgnoredType) {
						logger.Error().Err(err).Msg("failed to parse feature")
					}
					continue
				}

				sportsVenue.Source = fmt.Sprintf("%s/get/%d", sourceURL, feature.ID)

				attributes := convertDBSportsVenueToFiwareSportsVenue(*sportsVenue)

				fragment, _ := entities.NewFragment(attributes...)

				entityID := diwise.SportsVenueIDPrefix + sportsVenue.ID

				_, err = ctxBrokerClient.MergeEntity(ctx, entityID, fragment, headers)
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
	}

	return nil
}

func parsePublishedSportsVenue(log zerolog.Logger, feature domain.Feature) (*domain.SportsVenue, error) {
	log.Info().Msgf("found published sports venue %d %s", feature.ID, feature.Properties.Name)

	sportsVenue := &domain.SportsVenue{
		ID:          fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID),
		Name:        feature.Properties.Name,
		Description: "",
	}

	var timeFormat string = "2006-01-02 15:04:05"

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

	fields := []domain.FeaturePropField{}
	err = json.Unmarshal(feature.Properties.Fields, &fields)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal property fields %s: %s", string(feature.Properties.Fields), err.Error())
	}

	for _, field := range fields {
		if field.ID == 78 {
			sportsVenue.Description = string(field.Value[1 : len(field.Value)-1])
		} else if field.ID == 151 {
			url := string(field.Value[1 : len(field.Value)-1])
			url = strings.ReplaceAll(url, "\\/", "/")
			sportsVenue.SeeAlso = []string{url}
		}
	}

	// TODO: Fix these
	supportedCategories := map[string][]string{
		"Badhus":    {"swimming"},
		"Ishall":    {"ice-rink"},
		"Sporthall": {"sports"},
	}

	categories, ok := supportedCategories[feature.Properties.Type]
	if ok {
		sportsVenue.Category = categories
	}

	return sportsVenue, nil
}

func convertDBSportsVenueToFiwareSportsVenue(field domain.SportsVenue) []entities.EntityDecoratorFunc {

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 7),
		LocationMP(field.Geometry.Lines), Description(field.Description),
		DateTimeIfNotZero(properties.DateCreated, field.DateCreated),
		DateTimeIfNotZero(properties.DateModified, field.DateModified),
		Name(field.Name),
		Description(field.Description),
		TextList("seeAlso", field.SeeAlso),
	)

	if len(field.Category) > 0 {
		attributes = append(attributes, TextList("category", field.Category))
	}

	if field.Source != "" {
		attributes = append(attributes, Source(field.Source))
	}

	return attributes
}
