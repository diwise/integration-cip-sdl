package facilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
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

const (
	BikeTrail       string = "Cykelled"
	ExerciseTrail   string = "Motionsspår"
	IceSkatingTrail string = "Långfärdsskridskoled"
	SkiLift         string = "Skidlift"
	SkiSlope        string = "Skidpist"
	SkiTrack        string = "Skidspår"
)

func (s *storageImpl) StoreTrailsFromSource(ctx context.Context, ctxBrokerClient client.ContextBrokerClient, sourceURL string, featureCollection domain.FeatureCollection) error {

	logger := logging.GetFromContext(ctx)
	logger.Info("creating or updating exercise trails in broker...")

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	isSupportedType := func(theType string) bool {
		type StringSet map[string]struct{}
		_, theTypeIsInSet := StringSet{BikeTrail: {}, ExerciseTrail: {}, IceSkatingTrail: {}, SkiLift: {}, SkiSlope: {}, SkiTrack: {}}[theType]
		return theTypeIsInSet
	}

	for _, feature := range featureCollection.Features {
		if isSupportedType(feature.Properties.Type) {
			exerciseTrail, err := parseExerciseTrail(ctx, feature)
			if err != nil {
				logger.Error("failed to parse exercise trail", slog.Int64("featureID", feature.ID), "err", err.Error())
				continue
			}

			entityID := diwise.ExerciseTrailIDPrefix + exerciseTrail.ID

			if okToDel, alreadyDeleted := s.shouldBeDeleted(ctx, feature); okToDel {
				if !alreadyDeleted {
					_, err := ctxBrokerClient.DeleteEntity(ctx, entityID)
					if err != nil {
						logger.Info("could not delete entity", "entityID", entityID, "err", err.Error())
					}
				}
				continue
			}

			exerciseTrail.Source = fmt.Sprintf("%s/get/%d", sourceURL, feature.ID)

			attributes := convertDBTrailToFiwareExerciseTrail(*exerciseTrail)

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
				entity, err := entities.New(entityID, diwise.ExerciseTrailTypeName, attributes...)
				if err != nil {
					logger.Error("entities.New failed", "entityID", entityID, "err", err.Error())
					continue
				}

				deadline, cancelDeadline := context.WithDeadline(ctx, time.Now().Add(10*time.Second))
				_, err = ctxBrokerClient.CreateEntity(deadline, entity, headers)
				cancelDeadline()

				if err != nil {
					logger.Error("failed to post exercise trail to context broker", "entityID", entityID, "err", err.Error())
					continue
				}
			}
		}
	}

	logger.Info("done processing exercise trails")

	return nil
}

func parseExerciseTrail(ctx context.Context, feature domain.Feature) (*domain.ExerciseTrail, error) {
	log := logging.GetFromContext(ctx)
	log.Info("found published exercise trail", slog.Int64("featureID", feature.ID), slog.String("name", feature.Properties.Name))

	trail := &domain.ExerciseTrail{
		ID:          fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID),
		Name:        feature.Properties.Name,
		Description: "",
		Difficulty:  -1,
	}

	if !feature.Properties.Published {
		return trail, nil
	}

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

	if feature.Properties.Manager != nil {
		trail.ManagedBy = fmt.Sprintf("urn:ngsi-ld:Organisation:se:sundsvall:facilities:org:%d", feature.Properties.Manager.OrganisationID)
	}

	if feature.Properties.Owner != nil {
		trail.Owner = fmt.Sprintf("urn:ngsi-ld:Organisation:se:sundsvall:facilities:org:%d", feature.Properties.Owner.OrganisationID)
	}

	fields := []domain.FeaturePropField{}
	err = json.Unmarshal(feature.Properties.Fields, &fields)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal property fields %s: %s", string(feature.Properties.Fields), err.Error())
	}

	categories := []string{}

	if feature.Properties.Type == IceSkatingTrail {
		categories = append(categories, "ice-skating")
	} else if feature.Properties.Type == BikeTrail {
		categories = append(categories, "bike-track")
	} else if feature.Properties.Type == SkiSlope {
		categories = append(categories, "ski-slope")
	} else if feature.Properties.Type == SkiLift {
		categories = append(categories, "ski-lift")
	}

	for _, field := range fields {
		if field.ID == 99 {
			length, _ := strconv.ParseInt(string(field.Value[0:len(field.Value)]), 10, 64)
			trail.Length = float64(length) / 1000.0
		} else if field.ID == 100 {
			elevation, err := strconv.ParseInt(string(field.Value[0:len(field.Value)]), 10, 64)
			if err != nil {
				log.Error("failed to parse elevation gain on exercise trail", slog.String("name", trail.Name), slog.String("err", err.Error()))
			}
			trail.ElevationGain = float64(elevation)
		} else if field.ID == 102 {
			openStatus := map[string]string{"Ja": "open", "Nej": "closed"}
			trail.Status = openStatus[string(field.Value[1:len(field.Value)-1])]
		} else if field.ID == 103 {
			if propertyValueMatches(field, "Ja") {
				categories = append(categories, "floodlit")
			}
		} else if field.ID == 104 {
			avgift := string(field.Value[1 : len(field.Value)-1])
			if avgift != "Nej" {
				trail.PaymentRequired = true
			} else {
				trail.PaymentRequired = false
			}
		} else if field.ID == 109 {
			difficulty := map[string]float64{
				"Mycket lätt": 0,
				"Lätt":        1,
				"Medelsvår":   2,
				"Svår":        3,
				"Mycket svår": 4,
			}

			diff, ok := difficulty[string(field.Value[1:len(field.Value)-1])]
			if !ok {
				return nil, fmt.Errorf("difficulty level does not match known set")
			}
			trail.Difficulty = diff / 4.0
		} else if field.ID == 110 {
			trail.Description = string(field.Value[1 : len(field.Value)-1])
		} else if field.ID == 114 {
			trailTypes := map[string]string{
				"Crosscountry": "bike-track-xc",
				"Enduro":       "bike-track-enduro",
				"Flow":         "bike-track-flow",
			}

			trailType, ok := trailTypes[string(field.Value[1:len(field.Value)-1])]
			if ok {
				categories = append(categories, trailType)
			}
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
		} else if field.ID == 282 {
			publicAccess := map[string]string{
				"Hela dygnet":          "always",
				"Nej":                  "no",
				"Särskilda öppettider": "opening-hours",
				"Utanför skoltid":      "after-school",
			}
			paValue := string(field.Value[1 : len(field.Value)-1])

			var ok bool
			trail.PublicAccess, ok = publicAccess[paValue]
			if !ok {
				return nil, fmt.Errorf("unknown public access value: %s", paValue)
			}
		} else if field.ID == 283 {
			url := string(field.Value[1 : len(field.Value)-1])
			url = strings.ReplaceAll(url, "\\/", "/")
			trail.SeeAlso = []string{url}
		} else if field.ID == 284 {
			knownTypes := map[string]string{"Bygellift": "anchor-lift", "Knapplift": "button-lift"}
			if liftType, ok := knownTypes[string(field.Value[1:len(field.Value)-1])]; ok {
				categories = append(categories, liftType)
			}
		} else if field.ID == 294 {
			notes := strings.ReplaceAll(string(field.Value[1:len(field.Value)-1]), "\\/", "/")
			trail.Annotations = &notes
		} else if field.ID == 313 {
			widthValue, ok := strings.CutSuffix(string(field.Value[1:len(field.Value)-1]), " cm")
			if ok {
				trail.Width, err = strconv.ParseFloat(widthValue, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid trail width value \"%s\": %s", widthValue, err.Error())
				}
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

func convertDBTrailToFiwareExerciseTrail(trail domain.ExerciseTrail) []entities.EntityDecoratorFunc {

	boolMap := map[bool]string{
		true:  "yes",
		false: "no",
	}

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 21),
		LocationLS(trail.Geometry.Lines), Description(trail.Description),
		DateTimeIfNotZero(properties.DateCreated, trail.DateCreated),
		DateTimeIfNotZero(properties.DateModified, trail.DateModified),
		DateTimeIfNotZero("dateLastPreparation", trail.DateLastPrepared),
		Text("paymentRequired", boolMap[trail.PaymentRequired]),
		Name(trail.Name),
		Number("length", trail.Length),
		Description(trail.Description),
	)

	if trail.Annotations != nil {
		attributes = append(attributes, Text("annotations", *trail.Annotations))
	}

	if trail.AreaServed != "" {
		attributes = append(attributes, Text("areaServed", trail.AreaServed))
	}

	if len(trail.Category) > 0 {
		attributes = append(attributes, TextList("category", trail.Category))
	}

	if len(trail.PublicAccess) > 0 {
		attributes = append(attributes, Text("publicAccess", trail.PublicAccess))
	}

	if trail.Source != "" {
		attributes = append(attributes, Source(trail.Source))
	}

	if trail.Status != "" {
		attributes = append(attributes, Status(trail.Status))
	}

	if trail.Width > 0.1 {
		attributes = append(attributes, Number("width", math.Round(trail.Width*10)/10, properties.UnitCode("CMT")))
	}

	if trail.ManagedBy != "" {
		attributes = append(attributes, entities.R("managedBy", relationships.NewSingleObjectRelationship(trail.ManagedBy)))
	}

	if trail.Owner != "" {
		attributes = append(attributes, entities.R("owner", relationships.NewSingleObjectRelationship(trail.Owner)))
	}

	if trail.Difficulty >= 0 {
		// Add difficulty rounded to one decimal
		attributes = append(attributes, Number("difficulty", math.Round(trail.Difficulty*100)/100))
	}

	if trail.ElevationGain > 0.1 {
		attributes = append(attributes, Number("elevationGain", math.Round(trail.ElevationGain*10)/10, properties.UnitCode("MTR")))
	}

	if len(trail.SeeAlso) > 0 {
		attributes = append(attributes, TextList("seeAlso", trail.SeeAlso))
	}

	return attributes
}
