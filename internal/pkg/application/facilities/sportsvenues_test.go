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
)

const sportsVenueResponse string = `{"type":"FeatureCollection","features":[
	{"id":641,"type":"Feature","properties":{"name":"Stora ishallen","type":"Ishall","created":"2018-10-11 13:31:18","updated":"2021-02-19 14:42:28","published":true,
    "owner": {
      "organizationID": 170,
      "name": "sundsvalls kommun idrott och fritid"
    },
    "manager": {
      "organizationID": 888,
      "name": "sundsvalls kommun idrott och fritid"
    },"fields":[{"id":200,"name":"Allmänt tillgänglig","type":"DROPDOWN","value":"Särskilda öppettider"},{"id":96,"name":"Byggnad","type":"BUILDING","buildingStatus":"Gällande","buildingGuid":"a-long-and-unique-guid"},{"id":78,"name":"Beskrivning","type":"FREETEXT","value":"en bra beskrivning"},{"id":151,"name":"Öppettider länk","type":"FREETEXT","value":"https:\/\/sundsvall.se\/kontakter\/uthyrningsbyran-2\/"},{"id":152,"name":"Kontakt länk","type":"FREETEXT","value":"https:\/\/sundsvall.se\/kontakter\/uthyrningsbyran-2\/"}]},"geometry":{"type":"MultiPolygon","coordinates":[[[[17.34617972962255,62.412574010033595],[17.347279929404046,62.41262480545839],[17.34723895085804,62.41267442979932],[17.34677784825609,62.41265203299056],[17.3467384430584,62.412700741615886],[17.346158134543554,62.41267312155208],[17.34617972962255,62.412574010033595]]],[[[17.34761399162344,62.41237392319236],[17.34855035006534,62.41242025816159],[17.34855487518801,62.41240807636365],[17.348697258604084,62.41241502451384],[17.348694859337424,62.4124266585969],[17.3491824164106,62.41245060556396],[17.349186451844567,62.412439096546166],[17.34925259028114,62.412442947393956],[17.349242665669887,62.412488258989285],[17.34951570696293,62.41250324055892],[17.349450338564544,62.41279074442415],[17.34917837549993,62.41277769325007],[17.349174084398584,62.41278964661511],[17.34910750991809,62.41278567734849],[17.34911035207147,62.41277426029809],[17.348622494412524,62.41275099160957],[17.348618421809622,62.412762276725374],[17.34847512574829,62.41275531691869],[17.348477443178098,62.41274436672738],[17.347990639200116,62.412720198001196],[17.347987258641506,62.4127321629188],[17.347920447429903,62.4127286281349],[17.347926048373157,62.41270323609897],[17.347804278351486,62.41269643367649],[17.347780967699386,62.41264408493952],[17.34755669799009,62.412629526396316],[17.34761399162344,62.41237392319236]]],[[[17.345975603255734,62.412564444047085],[17.346071624928104,62.41211577666336],[17.34712002235073,62.412164188763754],[17.34713915061031,62.41208236135621],[17.347485282876978,62.412099515178376],[17.34746669392995,62.41218019533143],[17.347655476475186,62.41218891145522],[17.347554686956936,62.41263845980769],[17.345975603255734,62.412564444047085]]]]}}
]}`

func TestSportsVenueLoad(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, sportsVenueResponse)

	fc := domain.FeatureCollection{}
	json.Unmarshal([]byte(sportsVenueResponse), &fc)

	ctx := context.Background()
	storage := NewStorage(ctx)
	err := storage.StoreSportsVenuesFromSource(context.Background(), ctxBrokerMock, server.URL, fc)

	is.NoErr(err)
	is.Equal(len(ctxBrokerMock.MergeEntityCalls()), 1)
}

func TestSportsVenue(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, sportsVenueResponse)

	// Replace default failing CreateEntityFunc with a noop, so we can fetch the entity argument in the assert phase
	ctxBrokerMock.CreateEntityFunc = func(ctx context.Context, entity types.Entity, headers map[string][]string) (*ngsild.CreateEntityResult, error) {
		return &ngsild.CreateEntityResult{}, nil
	}

	client := NewClient("apiKey", server.URL)

	featureCollection, err := client.Get(context.Background())
	is.NoErr(err)

	ctx := context.Background()
	storage := NewStorage(ctx)
	err = storage.StoreSportsVenuesFromSource(context.Background(), ctxBrokerMock, server.URL, *featureCollection)
	is.NoErr(err)

	is.Equal(len(ctxBrokerMock.CreateEntityCalls()), 1)
	e := ctxBrokerMock.CreateEntityCalls()[0].Entity
	entityJSON, _ := json.Marshal(e)

	const name string = `"name":{"type":"Property","value":"Stora ishallen"}`
	is.True(strings.Contains(string(entityJSON), name))
}

func TestSportsVenueContainsManagedByAndOwnerProperties(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, sportsVenueResponse)

	// Replace default failing CreateEntityFunc with a noop, so we can fetch the entity argument in the assert phase
	ctxBrokerMock.CreateEntityFunc = func(ctx context.Context, entity types.Entity, headers map[string][]string) (*ngsild.CreateEntityResult, error) {
		return &ngsild.CreateEntityResult{}, nil
	}

	client := NewClient("apiKey", server.URL)

	ctx := context.Background()
	featureCollection, err := client.Get(ctx)
	is.NoErr(err)

	storage := NewStorage(ctx)
	err = storage.StoreSportsVenuesFromSource(ctx, ctxBrokerMock, server.URL, *featureCollection)
	is.NoErr(err)

	is.Equal(len(ctxBrokerMock.CreateEntityCalls()), 1)
	e := ctxBrokerMock.CreateEntityCalls()[0].Entity
	entityJSON, _ := json.Marshal(e)

	const manager string = `"managedBy":{"type":"Relationship","object":"urn:ngsi-ld:Organisation:se:sundsvall:facilities:org:888"}`
	const owner string = `"owner":{"type":"Relationship","object":"urn:ngsi-ld:Organisation:se:sundsvall:facilities:org:170"}`
	is.True(strings.Contains(string(entityJSON), manager))
	is.True(strings.Contains(string(entityJSON), owner))
}

func TestDeletedSportsVenue(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, sportsVenueResponse)

	fc := domain.FeatureCollection{}
	json.Unmarshal([]byte(sportsVenueResponse), &fc)

	var deletedDate = "2022-01-01 00:00:01"
	fc.Features[0].Properties.Deleted = &deletedDate

	ctx := context.Background()
	storage := NewStorage(ctx)
	err := storage.StoreSportsVenuesFromSource(ctx, ctxBrokerMock, server.URL, fc)
	is.NoErr(err)
	is.Equal(len(ctxBrokerMock.DeleteEntityCalls()), 1)
}
