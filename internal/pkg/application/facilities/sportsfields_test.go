package facilities

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/diwise/context-broker/pkg/ngsild"
	"github.com/diwise/context-broker/pkg/ngsild/types"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const sportsFieldResponse string = `{"type":"FeatureCollection","features":[
	{"id":796,"type":"Feature","properties":{"name":"Skolans grusplan och isbana","type":"Aktivitetsyta","created":"2019-10-15 16:15:32","updated":"2021-12-17 16:54:02","published":true,"owner":{"organizationID":168,"name":"sundsvalls kommun drakfastigheter"},"manager":{"organizationID":888,"name":"sundsvalls kommun idrott och fritid"},"tags":["7-mannaplan","bollplan","fotboll","grusplan","spontanyta"],"fields":[{"id":137,"name":"Isplan enklare","type":"TOGGLE","value":"Ja"},{"id":138,"name":"Ishockeyplan","type":"TOGGLE","value":"Nej"},{"id":139,"name":"Bandyplan","type":"TOGGLE","value":"Nej"},{"id":141,"name":"Fotbollsplan enklare","type":"TOGGLE","value":"Ja"},{"id":147,"name":"Sarg","type":"TOGGLE","value":"Nej"},{"id":154,"name":"Bokningsbar","type":"TOGGLE","value":"Nej"},{"id":156,"name":"Amerikansk fotbollsplan","type":"TOGGLE","value":"Nej"},{"id":157,"name":"Baseball eller softball","type":"TOGGLE","value":"Nej"},{"id":182,"name":"Rugby","type":"TOGGLE","value":"Nej"},{"id":225,"name":"Friidrott","type":"TOGGLE","value":"Nej"},{"id":279,"name":"Belysning","type":"TOGGLE","value":"Ja"},{"id":7,"name":"Tillhörande filer","type":"FILES","value":[{"id":2284,"filename":"Skolans isbana (1).jpg","description":"Isbana vid skolan","alttext":"Isbana med målburar","sourcetext":"Sundsvalls kommun","validForWinter":true,"validForSummer":false,"sortIndex":1,"type":"image\/jpeg","size":955471,"url":"https:\/\/anlaggning.sundsvall.se\/filesfield\/api\/2284"},{"id":548,"filename":"Skolans Grusplan 1c light.jpg","description":"Grusplan vid skolan","alttext":"Grusplan med målburar","sourcetext":"Sundsvalls kommun","validForWinter":false,"validForSummer":true,"sortIndex":2,"type":"image\/jpeg","size":1221415,"url":"https:\/\/anlaggning.sundsvall.se\/filesfield\/api\/548","license":"CC0","licenseDescription":"Creative commons. \"No rights reserved\" https:\/\/creativecommons.org\/share-your-work\/public-domain\/cc0\/"}]},{"id":34,"name":"Underlag","type":"DROPDOWN","value":"Grus"},{"id":153,"name":"Allmänt tillgänglig","type":"DROPDOWN","value":"Utanför skoltid"},{"id":1,"name":"Beskrivning","type":"FREETEXT","value":"7-manna grusplan intill skolan. Vintertid spolas och snöröjs isbanan en gång i veckan."},{"id":142,"name":"Fotbollsplan 5-manna antal","type":"INTEGER","value":1},{"id":143,"name":"Fotbollsplan 7-manna antal","type":"INTEGER","value":1}]},"geometry":{"type":"MultiPolygon","coordinates":[[[[17.428771593881844,62.42103804538807],[17.428785133659883,62.421037809376244],[17.428821575900738,62.42048396661722],[17.428101436027845,62.42046508568337],[17.428025378913084,62.42103219129709],[17.428365400350206,62.421045125144],[17.428690864217362,62.421045739009976],[17.428771593881844,62.42103804538807]]]]}}
	]}`

func TestSportsFieldLoad(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, sportsFieldResponse)

	fc := domain.FeatureCollection{}
	json.Unmarshal([]byte(sportsFieldResponse), &fc)

	err := StoreSportsFieldsFromSource(log.With().Logger(), ctxBrokerMock, context.Background(), server.URL, fc)
	is.NoErr(err)
	is.Equal(len(ctxBrokerMock.MergeEntityCalls()), 1)
}

func TestSportsField(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, sportsFieldResponse)

	// Replace default failing CreateEntityFunc with a noop, so we can fetch the entity argument in the assert phase
	ctxBrokerMock.CreateEntityFunc = func(ctx context.Context, entity types.Entity, headers map[string][]string) (*ngsild.CreateEntityResult, error) {
		return &ngsild.CreateEntityResult{}, nil
	}

	client := NewClient("apiKey", server.URL, zerolog.Logger{})

	featureCollection, err := client.Get(context.Background())
	is.NoErr(err)

	err = StoreSportsFieldsFromSource(zerolog.Logger{}, ctxBrokerMock, context.Background(), server.URL, *featureCollection)
	is.NoErr(err)

	is.Equal(len(ctxBrokerMock.CreateEntityCalls()), 1)
	e := ctxBrokerMock.CreateEntityCalls()[0].Entity
	entityJSON, _ := json.Marshal(e)

	const categories string = `"category":{"type":"Property","value":["skating","floodlit","ice-rink"]}`
	is.True(strings.Contains(string(entityJSON), categories))
}

func TestSportsFieldHasManagerAndOwnerProperties(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, sportsFieldResponse)

	// Replace default failing CreateEntityFunc with a noop, so we can fetch the entity argument in the assert phase
	ctxBrokerMock.CreateEntityFunc = func(ctx context.Context, entity types.Entity, headers map[string][]string) (*ngsild.CreateEntityResult, error) {
		return &ngsild.CreateEntityResult{}, nil
	}

	client := NewClient("apiKey", server.URL, zerolog.Logger{})

	featureCollection, err := client.Get(context.Background())
	is.NoErr(err)

	err = StoreSportsFieldsFromSource(zerolog.Logger{}, ctxBrokerMock, context.Background(), server.URL, *featureCollection)
	is.NoErr(err)

	is.Equal(len(ctxBrokerMock.CreateEntityCalls()), 1)
	e := ctxBrokerMock.CreateEntityCalls()[0].Entity
	entityJSON, _ := json.Marshal(e)

	const manager string = `"manager":{"type":"Relationship","object":"urn:ngsi-ld:Organisation:se:sundsvall:888"}`
	const owner string = `"owner":{"type":"Relationship","object":"urn:ngsi-ld:Organisation:se:sundsvall:168"}`
	is.True(strings.Contains(string(entityJSON), manager))
	is.True(strings.Contains(string(entityJSON), owner))
}
