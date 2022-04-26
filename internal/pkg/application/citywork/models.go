package citywork

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

type sdlResponse struct {
	Type string `json:"type"`
	Name string `json:"name"`
	CRS  struct {
		Type       string `json:"type"`
		Properties struct {
			Name string `json:"name"`
		} `json:"properties"`
	} `json:"crs"`
	Features []sdlFeature `json:"features"`
}

type sdlFeature struct {
	Type       string      `json:"type"`
	Geometry   sdlGeometry `json:"geometry"`
	Properties struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		Restrictions string `json:"restrictions"`
		Level        string `json:"level"`
		Start        string `json:"start"`
		End          string `json:"end"`
	} `json:"properties"`
}

type sdlGeometry struct {
	Type       string          `json:"type"`
	Geometries json.RawMessage `json:"geometries"`
}

func (sf *sdlFeature) ID() string {
	long, lat, _ := sf.Geometry.AsPoint()
	id := fmt.Sprintf("%b:%b:%s:%s", long, lat, sf.Properties.Start, sf.Properties.End)
	id = strings.ReplaceAll(strings.ReplaceAll(id, "-", ""), ".", "")
	return id
}

func (g *sdlGeometry) AsPoint() (float64, float64, error) {
	temp := []struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}{}

	err := json.Unmarshal(g.Geometries, &temp)
	if err != nil {
		return 0, 0, err
	}

	for _, c := range temp {
		if c.Type == "Point" {
			var p []float64
			err = json.Unmarshal(c.Coordinates, &p)
			if err != nil {
				return 0, 0, err
			}
			
			x, y := convertSWEREFtoWGS84(p[1], p[0])
			
			return x, y, nil
		}
	}

	return 0, 0, fmt.Errorf("unable to parse point")
}

func convertSWEREFtoWGS84(x, y float64) (float64, float64) {

	//Code adapted from
	//https://github.com/bjornsallarp/MightyLittleGeodesy/blob/master/MightyLittleGeodesy/Classes/GaussKreuger.cs

	var axis float64 = 6378137.0                 // GRS 80.
	var flattening float64 = 1.0 / 298.257222101 // GRS 80.

	var centralMeridian float64 = 15.00
	var scale float64 = 0.9996
	var falseNorthing float64 = 0.0
	var falseEasting float64 = 500000.0

	e2 := flattening * (2.0 - flattening)
	n := flattening / (2.0 - flattening)

	aRoof := axis / (1.0 + n) * (1.0 + n*n/4.0 + n*n*n*n/64.0)
	delta1 := n/2.0 - 2.0*n*n/3.0 + 37.0*n*n*n/96.0 - n*n*n*n/360.0
	delta2 := n*n/48.0 + n*n*n/15.0 - 437.0*n*n*n*n/1440.0
	delta3 := 17.0*n*n*n/480.0 - 37*n*n*n*n/840.0
	delta4 := 4397.0 * n * n * n * n / 161280.0

	Astar := e2 + e2*e2 + e2*e2*e2 + e2*e2*e2*e2
	Bstar := -(7.0*e2*e2 + 17.0*e2*e2*e2 + 30.0*e2*e2*e2*e2) / 6.0
	Cstar := (224.0*e2*e2*e2 + 889.0*e2*e2*e2*e2) / 120.0
	Dstar := -(4279.0 * e2 * e2 * e2 * e2) / 1260.0

	// Convert.
	degToRad := math.Pi / 180
	lambdaZero := centralMeridian * degToRad
	xi := (x - falseNorthing) / (scale * aRoof)
	eta := (y - falseEasting) / (scale * aRoof)
	xiPrim := xi -
		delta1*math.Sin(2.0*xi)*math.Cosh(2.0*eta) -
		delta2*math.Sin(4.0*xi)*math.Cosh(4.0*eta) -
		delta3*math.Sin(6.0*xi)*math.Cosh(6.0*eta) -
		delta4*math.Sin(8.0*xi)*math.Cosh(8.0*eta)
	etaPrim := eta -
		delta1*math.Cos(2.0*xi)*math.Sinh(2.0*eta) -
		delta2*math.Cos(4.0*xi)*math.Sinh(4.0*eta) -
		delta3*math.Cos(6.0*xi)*math.Sinh(6.0*eta) -
		delta4*math.Cos(8.0*xi)*math.Sinh(8.0*eta)

	phiStar := math.Asin(math.Sin(xiPrim) / math.Cosh(etaPrim))
	deltaLambda := math.Atan(math.Sinh(etaPrim) / math.Cos(xiPrim))

	lonRadian := lambdaZero + deltaLambda
	latRadian := phiStar + math.Sin(phiStar)*math.Cos(phiStar)*
		(Astar+
			Bstar*math.Pow(math.Sin(phiStar), 2)+
			Cstar*math.Pow(math.Sin(phiStar), 4)+
			Dstar*math.Pow(math.Sin(phiStar), 6))

	lat := latRadian * 180.0 / math.Pi
	lon := lonRadian * 180.0 / math.Pi

	return lon, lat
}