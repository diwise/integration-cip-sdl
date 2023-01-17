package facilities

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/diwise/context-broker/pkg/ngsild"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/rs/zerolog"
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

func TestDeletedBeach(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, response)

	ctxBrokerMock.DeleteEntityFunc = func(ctx context.Context, entityID string) (*ngsild.DeleteEntityResult, error) {
		return &ngsild.DeleteEntityResult{}, nil
	}

	client := NewClient("apiKey", server.URL, zerolog.Logger{})

	featureCollection, err := client.Get(context.Background())
	is.NoErr(err)

	var deletedDate = "2022-01-01 00:00:01"
	featureCollection.Features[0].Properties.Deleted = &deletedDate

	err = StoreBeachesFromSource(zerolog.Logger{}, ctxBrokerMock, context.Background(), server.URL, *featureCollection)
	is.NoErr(err)

	is.Equal(len(ctxBrokerMock.DeleteEntityCalls()), 1)
}
