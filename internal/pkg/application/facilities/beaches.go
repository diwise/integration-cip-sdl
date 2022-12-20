package facilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/diwise/context-broker/pkg/datamodels/fiware"
	"github.com/diwise/context-broker/pkg/ngsild/client"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/rs/zerolog"
)

func StoreBeachesFromSource(logger zerolog.Logger, ctxBrokerClient client.ContextBrokerClient, ctx context.Context, sourceURL string, featureCollection domain.FeatureCollection) error {
	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	for _, feature := range featureCollection.Features {
		if feature.Properties.Type == "Strandbad" {
			beach, published, err := parseBeach(logger, feature)
			if err != nil {
				logger.Error().Err(err).Msg("failed to parse strandbad")
				continue
			}

			entityID := fiware.BeachIDPrefix + beach.ID

			if !published {
				if shouldBeDeleted(feature) {
					_, err := ctxBrokerClient.DeleteEntity(ctx, entityID)
					if err != nil {
						logger.Info().Msgf("could not delete entity %s", entityID)
					}
				}
				continue
			}

			attributes := convertDomainBeachToFiwareBeach(*beach)

			fragment, _ := entities.NewFragment(attributes...)

			_, err = ctxBrokerClient.MergeEntity(ctx, entityID, fragment, headers)

			// Throttle so we dont kill the broker
			time.Sleep(500 * time.Millisecond)

			if err != nil {
				if !errors.Is(err, ngsierrors.ErrNotFound) {
					logger.Error().Err(err).Msg("failed to merge entity")
					continue
				}
				entity, err := entities.New(entityID, fiware.BeachTypeName, attributes...)
				if err != nil {
					logger.Error().Err(err).Msg("entities.New failed")
					continue
				}

				res, err := ctxBrokerClient.CreateEntity(ctx, entity, headers)
				if err != nil {
					logger.Error().Err(err).Msg("failed to post beach to context broker")
					continue
				}

				logger.Info().Msgf("posted beach %s to context broker", res.Location())
			}
		}

	}

	return nil
}

func parseBeach(log zerolog.Logger, feature domain.Feature) (*domain.Beach, bool, error) {
	if feature.Properties.Published {
		b, err := parsePublishedBeach(log, feature)
		return b, true, err
	}
	
	beach := &domain.Beach{
		ID:          fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID),
		Name:        feature.Properties.Name,
		Description: "",
	}

	return beach, false, nil
}

func parsePublishedBeach(log zerolog.Logger, feature domain.Feature) (*domain.Beach, error) {
	log.Info().Msgf("found published beach %d %s\n", feature.ID, feature.Properties.Name)

	beach := &domain.Beach{
		ID:          fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID),
		Name:        feature.Properties.Name,
		Description: "",
	}

	if feature.Properties.Created != nil {
		created, err := time.Parse(timeFormat, *feature.Properties.Created)
		if err == nil {
			beach.DateCreated = created.UTC()
		}
	}

	if feature.Properties.Updated != nil {	beach := &domain.Beach{
		ID:          fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID),
		Name:        feature.Properties.Name,
		Description: "",
	}
		modified, err := time.Parse(timeFormat, *feature.Properties.Updated)
		if err == nil {
			beach.DateModified = modified.UTC()
		}
	}

	err := json.Unmarshal(feature.Geometry.Coordinates, &beach.Geometry.Lines)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal geometry %s: %s", string(feature.Geometry.Coordinates), err.Error())
	}

	fields := []domain.FeaturePropField{}
	err = json.Unmarshal(feature.Properties.Fields, &fields)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal property fields %s: %s", string(feature.Properties.Fields), err.Error())
	}

	for _, field := range fields {
		if field.ID == 1 {
			beach.Description = string(field.Value[1 : len(field.Value)-1])
		} else if field.ID == 230 {
			sensor := "se:servanet:lora:" + string(field.Value[1:len(field.Value)-1])
			beach.SensorID = &sensor
			log.Info().Msgf("assigning sensor %s to beach %s", sensor, beach.ID)
		}
	}

	if ref, ok := seeAlsoRefs[feature.ID]; ok {
		if len(ref.nuts) > 0 {
			beach.NUTSCode = &ref.nuts
		}

		if len(ref.wikidata) > 0 {
			beach.WikidataID = &ref.wikidata
		}
	}

	return beach, nil
}

type extraInfo struct {
	nuts     string
	wikidata string
	sensorID string
}

var seeAlsoRefs map[int64]extraInfo = map[int64]extraInfo{
	// Slädaviken
	283: {nuts: "SE0712281000003473", sensorID: "sk-elt-temp-21", wikidata: "Q10671745"},
	// Hartungviken
	284: {nuts: "SE0712281000003472", sensorID: "sk-elt-temp-28", wikidata: "Q680645"},
	// Tranviken
	295: {nuts: "SE0712281000003474", sensorID: "sk-elt-temp-22", wikidata: "Q106657132"},
	// Bänkåsviken
	315: {nuts: "SE0712281000003471", sensorID: "sk-elt-temp-26", wikidata: "Q106657054"},
	// Stekpannan, Hornsjön
	322: {nuts: "SE0712281000003478", sensorID: "sk-elt-temp-17", wikidata: "Q106710721"},
	// Dyket
	323: {nuts: "SE0712281000003477", sensorID: "sk-elt-temp-02", wikidata: "Q106710719"},
	// Fläsian, Nord
	337: {nuts: "SE0712281000003450", sensorID: "sk-elt-temp-25"},
	// Sodom
	357: {nuts: "SE0712281000003479", sensorID: "sk-elt-temp-27", wikidata: "Q106710722"},
	// Rännö
	414: {nuts: "SE0712281000003464", sensorID: "sk-elt-temp-08", wikidata: "Q106710690"},
	// Lucksta
	421: {nuts: "SE0712281000003461", sensorID: "sk-elt-temp-10", wikidata: "Q106710684"},
	// Norrhassel
	430: {nuts: "SE0712281000003462", sensorID: "sk-elt-temp-13", wikidata: "Q106710685"},
	// Viggesand
	442: {nuts: "SE0712281000003469", sensorID: "sk-elt-temp-12", wikidata: "Q106710700"},
	// Räveln
	456: {nuts: "SE0712281000003468", sensorID: "sk-elt-temp-19", wikidata: "Q106710698"},
	// Segersjön
	469: {nuts: "SE0712281000003452", sensorID: "sk-elt-temp-09", wikidata: "Q106710670"},
	// Vången
	488: {nuts: "SE0712281000003470", sensorID: "sk-elt-temp-16", wikidata: "Q106710701"},
	// Edeforsens badplats
	495: {nuts: "SE0712281000003467", sensorID: "sk-elt-temp-04", wikidata: "Q106710696"},
	// Pallviken
	513: {nuts: "SE0712281000003463", sensorID: "sk-elt-temp-11", wikidata: "Q106710688"},
	// Östtjärn
	526: {nuts: "SE0712281000003466", sensorID: "sk-elt-temp-18", wikidata: "Q106710694"},
	// Bergafjärden
	553: {nuts: "SE0712281000003475", sensorID: "sk-elt-temp-24", wikidata: "Q16498519"},
	// Brudsjön
	560: {nuts: "SE0712281000003455", sensorID: "sk-elt-temp-03", wikidata: "Q106710675"},
	// Sandnäset
	656: {nuts: "SE0712281000003459", sensorID: "sk-elt-temp-14", wikidata: "Q106710678"},
	657: {sensorID: "sk-elt-temp-07"}, // Abborrviken, Sidsjön
	// Västbyn
	658: {nuts: "SE0712281000003460", sensorID: "sk-elt-temp-15", wikidata: "Q106710681"},
	// Väster-Lövsjön
	659: {nuts: "SE0712281000003453", sensorID: "sk-elt-temp-05", wikidata: "Q106710672"},
	// Sidsjöns hundbad
	660: {nuts: "SE0712281000004229", sensorID: "sk-elt-temp-01"},
	// Kävstabadet, Indal
	897: {nuts: "SE0712281000003456", wikidata: "Q106710677"},
	// Bredsand
	1234: {nuts: "SE0712281000003476", sensorID: "sk-elt-temp-23", wikidata: "Q106710717"},
	// Bjässjön
	1618: {nuts: "SE0712281000003454", sensorID: "sk-elt-temp-06", wikidata: "Q106947945"},
	// Fläsian, Syd
	1631: {nuts: "SE0712281000003480", sensorID: "sk-elt-temp-20"},
}

func convertDomainBeachToFiwareBeach(b domain.Beach) []entities.EntityDecoratorFunc {

	properties := []entities.EntityDecoratorFunc{
		entities.DefaultContext(),
		decorators.Description(b.Description),
		decorators.LocationMP(b.Geometry.Lines),
		decorators.DateTimeIfNotZero(properties.DateCreated, b.DateCreated),
		decorators.DateTimeIfNotZero(properties.DateModified, b.DateModified),
		decorators.Name(b.Name),
	}

	if b.SensorID != nil {
		references := []string{fmt.Sprintf("%s%s", fiware.DeviceIDPrefix, *b.SensorID)}
		properties = append(properties, decorators.RefSeeAlso(references))
	}

	seeAlso := []string{}

	if b.NUTSCode != nil {
		seeAlso = append(seeAlso, fmt.Sprintf("https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/%s", *b.NUTSCode))
	}

	if b.WikidataID != nil {
		seeAlso = append(seeAlso, fmt.Sprintf("https://www.wikidata.org/wiki/%s", *b.WikidataID))
	}

	if len(seeAlso) > 0 {
		properties = append(properties, decorators.TextList("seeAlso", seeAlso))
	}

	return properties
}
