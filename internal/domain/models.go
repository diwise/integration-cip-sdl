package domain

import (
	"encoding/json"
	"time"
)

type LineString struct {
	Lines [][]float64
}

type MultiPolygon struct {
	Lines [][][][]float64
}

// Beach contains a point of interest of type Beach
type Beach struct {
	ID               string
	Name             string
	Description      string
	Geometry         MultiPolygon
	WikidataID       *string
	NUTSCode         *string
	SensorID         *string
	WaterTemperature *float64
	DateCreated      time.Time
	DateModified     time.Time
}

// ExerciseTrail contains a point of interest of type ExerciseTrail
type ExerciseTrail struct {
	ID               string
	Name             string
	Description      string
	Category         []string
	Length           float64
	AreaServed       string
	Geometry         LineString
	Status           string
	DateCreated      time.Time
	DateModified     time.Time
	DateLastPrepared time.Time
	Source           string
	Difficulty       float64
	PaymentRequired  bool
	MapURL           string
}

// ---

const (
	SundsvallAnlaggningPrefix string = "se:sundsvall:facilities:"
)

type FeatureGeom struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type FeaturePropField struct {
	ID    int64           `json:"id"`
	Value json.RawMessage `json:"value"`
}

type FeatureProps struct {
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Published bool            `json:"published"`
	Fields    json.RawMessage `json:"fields"`
	Created   *string         `json:"created,omitempty"`
	Updated   *string         `json:"updated,omitempty"`
}

type Feature struct {
	ID         int64        `json:"id"`
	Properties FeatureProps `json:"properties"`
	Geometry   FeatureGeom  `json:"geometry"`
}

type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}
