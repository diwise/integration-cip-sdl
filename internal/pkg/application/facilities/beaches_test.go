package facilities

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
)

func TestBeachesDataLoad(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, response)
	ctx := context.Background()

	fc := domain.FeatureCollection{}
	json.Unmarshal([]byte(response), &fc)

	storage := NewStorage(ctx)
	err := storage.StoreBeachesFromSource(ctx, ctxBrokerMock, server.URL, fc)

	is.NoErr(err)
	is.Equal(len(ctxBrokerMock.CreateEntityCalls()), 1)
}

func TestDeletedBeach(t *testing.T) {
	is, ctxBrokerMock, server := testSetup(t, "", http.StatusOK, response)
	ctx := context.Background()

	ctxBrokerMock.DeleteEntityFunc = func(ctx context.Context, entityID string) (*ngsild.DeleteEntityResult, error) {
		return &ngsild.DeleteEntityResult{}, nil
	}

	client := NewClient(ctx, "apiKey", server.URL)

	featureCollection, err := client.Get(ctx)
	is.NoErr(err)

	var aWeekAgo = time.Now().UTC().Add(-1 * 7 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	featureCollection.Features[0].Properties.Deleted = &aWeekAgo

	storage := NewStorage(ctx)
	err = storage.StoreBeachesFromSource(ctx, ctxBrokerMock, server.URL, *featureCollection)

	is.NoErr(err)
	is.Equal(len(ctxBrokerMock.DeleteEntityCalls()), 1)
}
