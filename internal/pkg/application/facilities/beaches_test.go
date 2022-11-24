package facilities

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/rs/zerolog/log"
)

func TestBeachesDataLoad(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, response)

	fc := domain.FeatureCollection{}
	json.Unmarshal([]byte(response), &fc)

	err := StoreBeachesFromSource(log.With().Logger(), ctxBrokerMock, context.Background(), server.URL, fc)
	is.NoErr(err)
	is.Equal(len(ctxBrokerMock.CreateEntityCalls()), 1)
}
