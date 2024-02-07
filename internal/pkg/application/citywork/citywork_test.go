package citywork

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/context-broker/pkg/ngsild"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types"
	test "github.com/diwise/context-broker/pkg/test"
	"github.com/matryer/is"
)

func TestThatMergeIsAttemptedBeforeCreate(t *testing.T) {
	is, cw, ctxBroker := testSetup(t, 200, complex)

	err := cw.getAndPublishCityWork(context.Background())
	is.NoErr(err)
	is.Equal(len(ctxBroker.MergeEntityCalls()), 2)
	is.Equal(len(ctxBroker.CreateEntityCalls()), 2)
}

func TestSimpleModelCanBeCreated(t *testing.T) {
	is, _, _ := testSetup(t, 0, "")
	m, err := toModel([]byte(simple))

	is.NoErr(err)
	is.True(m != nil)
}

func TestComplexModelCanBeCreated(t *testing.T) {
	is, _, _ := testSetup(t, 0, "")
	m, _ := toModel([]byte(complex))

	long, lat, err := m.Features[0].Geometry.AsPoint()

	is.Equal(long, 17.202583472441642)
	is.Equal(lat, 62.397368375410174)

	is.NoErr(err)
	is.True(m != nil)
}

func TestThatGetAndPublishWorksWithSimpleResponse(t *testing.T) {
	is, cw, _ := testSetup(t, http.StatusOK, simple)

	err := cw.getAndPublishCityWork(context.Background())
	is.NoErr(err)
}

func TestThatGetAndPublishWorksWithComplexResponse(t *testing.T) {
	is, cw, _ := testSetup(t, http.StatusOK, complex)

	err := cw.getAndPublishCityWork(context.Background())
	is.NoErr(err)
}

func TestThatGetAndPublishFailsOnInternalServerError(t *testing.T) {
	is, cw, _ := testSetup(t, http.StatusInternalServerError, "")

	err := cw.getAndPublishCityWork(context.Background())
	is.True(err != nil)
}

func TestThatGetAndPublishFailsOnImproperJSON(t *testing.T) {
	is, cw, _ := testSetup(t, http.StatusOK, complex+"}")

	err := cw.getAndPublishCityWork(context.Background())
	is.True(err != nil)
}

func testSetup(t *testing.T, statusCode int, body string) (*is.I, CityWorkSvc, *test.ContextBrokerClientMock) {
	is := is.New(t)
	s := setupMockServiceThatReturns(statusCode, body)
	sdlc := sdlClient{
		sundsvallvaxerURL: s.URL,
	}

	ctxBroker := &test.ContextBrokerClientMock{
		CreateEntityFunc: func(ctx context.Context, entity types.Entity, headers map[string][]string) (*ngsild.CreateEntityResult, error) {
			return &ngsild.CreateEntityResult{}, fmt.Errorf("not implemented")
		},
		MergeEntityFunc: func(ctx context.Context, entityID string, fragment types.EntityFragment, headers map[string][]string) (*ngsild.MergeEntityResult, error) {
			return nil, ngsierrors.ErrNotFound
		},
	}

	cw := NewCityWorkService(context.Background(), &sdlc, 1, ctxBroker)

	return is, cw, ctxBroker
}

func setupMockServiceThatReturns(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
}

func toModel(resp []byte) (*sdlResponse, error) {
	var m sdlResponse

	err := json.Unmarshal(resp, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

const simple string = `
{
    "type": "FeatureCollection",
    "name": "Sundsvall Växer trafikstörningar",
    "crs": {
        "type": "name",
        "properties": {
            "name": "urn:ogc:def:crs:EPSG::3006"
        }
    },
	"features": [
        {
            "type": "Feature",
            "geometry": {
                "type": "GeometryCollection",
                "geometries": [{
                    "type": "Point",
                    "coordinates": [620761.3999999999, 6922510.664999999]
                }]
            },
            "properties": {
                "disruptionID": 490,
                "title": "Begränsad framkomlighet Norbergsvägen",
                "description": "I samband med schakt för ny elservis råder begränsad framkomlighet vid Norbergsvägen 6-8. Ett körfält kommer vara stängt förbi arbetsområdet.\r\n\r\nKontaktperson;\r\nSundsvall Energi\r\nUrban Tellebo\r\nurban.tellebo@sundsvallelnat.se\r\nTel: 073-2761221",
                "restrictions": null,
                "level": "SMALL",
                "disruptionStart": "2022-05-01Z",
                "disruptionEnd": "2022-06-29Z"
            }
        },
        {
            "type": "Feature",
            "geometry": {
                "type": "GeometryCollection",
                "geometries": [{
                    "type": "Point",
                    "coordinates": [619615.359991455, 6925559.199923094]
                }]
            },
            "properties": {
                "disruptionID": 471,
                "title": "Begränsad framkomlighet Arbetsledarvägen",
                "description": "I samband med schakt för nyanslutning av el vid Arbetsledarvägen 2-8 råder begränsad framkomlighet. Ett körfält kommer vara öppet förbi arbetsområdet.\r\n\r\nKontaktperson;\r\nSundsvall Elnät AB\r\nUrban Thellebo\r\nurban.thellebo@sundsvallelnat.se\r\nTel: 0606005000",
                "restrictions": null,
                "level": "SMALL",
                "disruptionStart": "2022-01-23Z",
                "disruptionEnd": "2022-05-31Z"
            }
        }
	]
}
`

const complex string = `
{
    "type": "FeatureCollection",
    "name": "Sundsvall Växer trafikstörningar",
    "crs": {
        "type": "name",
        "properties": {
            "name": "urn:ogc:def:crs:EPSG::3006"
        }
    },
	"features": [
        {
            "type": "Feature",
            "geometry": {
                "type": "GeometryCollection",
                "geometries": [{
                    "type": "Point",
                    "coordinates": [613844, 6920388.159927368]
                }, {
                    "type": "Polygon",
                    "coordinates": [
                        [
                            [610696.8, 6918052.231996154],
                            [611149, 6918183.159980773],
                            [611765, 6918526.159980773],
                            [612185, 6918820.159980773],
                            [612479, 6919086.159980773],
                            [612675, 6919317.159980773],
                            [612948, 6919478.159980773],
                            [613228, 6919737.159980773],
                            [613578, 6920066.159980773],
                            [613746, 6920276.159980773],
                            [614131, 6920416.159980773],
                            [614649, 6920626.159980773],
                            [614845, 6920710.159980773],
                            [615132, 6920731.159980773],
                            [615391, 6920724.159980773],
                            [615559, 6920661.159980773],
                            [615867, 6920570.159980773],
                            [615895, 6920654.159980773],
                            [615412, 6920787.159980773],
                            [614803, 6920787.159980773],
                            [613704, 6920367.159980773],
                            [612892, 6919576.159980773],
                            [612491.6, 6919286.86399231],
                            [612242.3999999999, 6918945.26399231],
                            [611911.9999999999, 6918696.063992309],
                            [611494.7999999998, 6918432.86399231],
                            [611049.6000000001, 6918208.86399231],
                            [610688.4, 6918105.26399231],
                            [610696.8, 6918052.231996154]
                        ]
                    ]
                }]
            },
            "properties": {
                "disruptionID": 5,
                "title": "Sprängarbeten på E14",
                "description": "Mellan den 25 april 2019 och 28 februari 2020 utför Trafikverkets entreprenör sprängarbeten längs E14, mellan cirkulationsplatsen Timmervägen/E14 och Blåberget.",
                "restrictions": "Det innebär kortare stopp i trafiken en till två gånger om dagen under perioden.",
                "level": "LARGE",
                "disruptionStart": "2019-04-24Z",
                "disruptionEnd": "2021-12-30Z"
            }
        },
		{
            "type": "Feature",
            "geometry": {
                "type": "GeometryCollection",
                "geometries": [{
                    "type": "Point",
                    "coordinates": [619615.359991455, 6925559.199923094]
                }]
            },
            "properties": {
                "disruptionID": 471,
                "title": "Begränsad framkomlighet Arbetsledarvägen",
                "description": "I samband med schakt för nyanslutning av el vid Arbetsledarvägen 2-8 råder begränsad framkomlighet. Ett körfält kommer vara öppet förbi arbetsområdet.\r\n\r\nKontaktperson;\r\nSundsvall Elnät AB\r\nUrban Thellebo\r\nurban.thellebo@sundsvallelnat.se\r\nTel: 0606005000",
                "restrictions": null,
                "level": "SMALL",
                "disruptionStart": "2022-01-23Z",
                "disruptionEnd": "2022-05-31Z"
            }
        }
	]
}
`
