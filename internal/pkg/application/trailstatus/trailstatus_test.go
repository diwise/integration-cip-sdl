package trailstatus

import (
	"context"
	"testing"

	"github.com/diwise/integration-cip-sdl/internal/domain"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestThatUpdateTrailStatusFromSourceCallsAddEntity(t *testing.T) {
	is, ts := testSetup(t)

	err := ts.UpdateTrailStatusFromSource(context.Background())
	is.NoErr(err)
}

func testSetup(t *testing.T) (*is.I, TrailPreparationService) {
	is := is.New(t)
	mcb := &domain.ContextBrokerClientMock{}
	ts := NewTrailPreparationService(zerolog.Logger{}, nil, mcb, "url")

	return is, ts
}

//get some of that Ski json in here
