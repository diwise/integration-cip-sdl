package citywork

import (
	"encoding/json"
	"testing"

	"github.com/matryer/is"
)

func TestSimpleModelCanBeCreated(t *testing.T) {
	is := testSetup(t)
	m, err := toModel([]byte(simple))

	is.True(m != nil)
	is.NoErr(err)
}

func TestComplexModelCanBeCreated(t *testing.T) {
	is := testSetup(t)
	m, _ := toModel([]byte(complex))

	long, lat, err := m.Features[0].Geometry.AsPoint()

	is.Equal(long, 17.202583472441642)
	is.Equal(lat, 62.397368375410174)

	is.True(m != nil)
	is.NoErr(err)
}

func TestModelCanBeConvertedToCityWork(t *testing.T) {
	is := testSetup(t)
	m, _ := toModel([]byte(simple))

	cw := toCityWorkModel(m.Features[0])

	is.Equal(cw.ID, "urn:ngsi-ld:CityWork:4905302560875326p48:8785065399290166p47:20220420:20220531")
}

func toModel(resp []byte) (*sdlResponse, error) {
	var m sdlResponse

	err := json.Unmarshal(resp, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func testSetup(t *testing.T) *is.I {
	is := is.New(t)
	return is
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
				"geometries": [
					{
						"type": "Point",
						"coordinates": [
							625345.0000000001,
							6923506.905000001
						]
					}
				]
			},
			"properties": {
				"title": "Gång- och cykelväg Kryssarvägen/Barkassvägen",
				"description": "<p>I samband med byte av nätstation på Kryssarvägen kommer gång- och cykelvägen att ledas om. En tillfällig gång- och cykelväg anläggs bredvid arbetsområdet. Följ orange hänvisning. </p><p>Kontaktperson;</p><p>Kontaktkort</p><p>Sundsvall Elnät AB</p><p>Urban Thellebo</p><p>urban.thellebo@sundsvallelnat.se</p><p>Tel: 0606005000</p>",
				"level": "SMALL",
				"start": "2022-04-20",
				"end": "2022-05-31"
			}
		},
		{
			"type": "Feature",
			"geometry": {
				"type": "GeometryCollection",
				"geometries": [
					{
						"type": "Point",
						"coordinates": [
							610950.7600000001,
							6928480.825
						]
					}
				]
			},
			"properties": {
				"title": "Begränsad framkomlighet Ånäsvägen/Viljansvägen",
				"description": "<p>I samband med fiberanslutning till Ånäsvägen 1 & 4 råder begränsad framkomlighet. Gång- och cykelvägen kommer vara framkomlig under arbetstiden.</p><p>Kontaktperson;</p><p>KME</p><p>Roger ytterström</p><p>roger@kme.se</p><p>Tel: 0705591062</p>",
				"level": "SMALL",
				"start": "2022-04-21",
				"end": "2022-05-31"
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
                "geometries": [
                    {
                        "type": "Point",
                        "coordinates": [
                            613844,
                            6920388.159927368
                        ]
                    },
                    {
                        "type": "Polygon",
                        "coordinates": [
                            [
                                [
                                    610696.8,
                                    6918052.231996154
                                ],
                                [
                                    611149,
                                    6918183.159980773
                                ],
                                [
                                    611765,
                                    6918526.159980773
                                ],
                                [
                                    612185,
                                    6918820.159980773
                                ],
                                [
                                    612479,
                                    6919086.159980773
                                ],
                                [
                                    612675,
                                    6919317.159980773
                                ],
                                [
                                    612948,
                                    6919478.159980773
                                ],
                                [
                                    613228,
                                    6919737.159980773
                                ],
                                [
                                    613578,
                                    6920066.159980773
                                ],
                                [
                                    613746,
                                    6920276.159980773
                                ],
                                [
                                    614131,
                                    6920416.159980773
                                ],
                                [
                                    614649,
                                    6920626.159980773
                                ],
                                [
                                    614845,
                                    6920710.159980773
                                ],
                                [
                                    615132,
                                    6920731.159980773
                                ],
                                [
                                    615391,
                                    6920724.159980773
                                ],
                                [
                                    615559,
                                    6920661.159980773
                                ],
                                [
                                    615867,
                                    6920570.159980773
                                ],
                                [
                                    615895,
                                    6920654.159980773
                                ],
                                [
                                    615412,
                                    6920787.159980773
                                ],
                                [
                                    614803,
                                    6920787.159980773
                                ],
                                [
                                    613704,
                                    6920367.159980773
                                ],
                                [
                                    612892,
                                    6919576.159980773
                                ],
                                [
                                    612491.6,
                                    6919286.86399231
                                ],
                                [
                                    612242.3999999999,
                                    6918945.26399231
                                ],
                                [
                                    611911.9999999999,
                                    6918696.063992309
                                ],
                                [
                                    611494.7999999998,
                                    6918432.86399231
                                ],
                                [
                                    611049.6000000001,
                                    6918208.86399231
                                ],
                                [
                                    610688.4,
                                    6918105.26399231
                                ],
                                [
                                    610696.8,
                                    6918052.231996154
                                ]
                            ]
                        ]
                    }
                ]
            },
            "properties": {
                "title": "Sprängarbeten på E14",
                "description": "<p>Mellan den 25 april 2019 och 28 februari 2020 utför Trafikverkets entreprenör sprängarbeten längs E14, mellan cirkulationsplatsen Timmervägen/E14 och Blåberget.</p>",
                "restrictions": "<p>Det innebär kortare stopp i trafiken en till två gånger om dagen under perioden.</p>",
                "level": "LARGE",
                "start": "2019-04-25",
                "end": "2021-12-31"
            }
        },
		{
			"type": "Feature",
			"geometry": {
				"type": "GeometryCollection",
				"geometries": [
					{
						"type": "Point",
						"coordinates": [
							610950.7600000001,
							6928480.825
						]
					}
				]
			},
			"properties": {
				"title": "Begränsad framkomlighet Ånäsvägen/Viljansvägen",
				"description": "<p>I samband med fiberanslutning till Ånäsvägen 1 & 4 råder begränsad framkomlighet. Gång- och cykelvägen kommer vara framkomlig under arbetstiden.</p><p>Kontaktperson;</p><p>KME</p><p>Roger ytterström</p><p>roger@kme.se</p><p>Tel: 0705591062</p>",
				"level": "SMALL",
				"start": "2022-04-21",
				"end": "2022-05-31"
			}
		}
	]
}
`
