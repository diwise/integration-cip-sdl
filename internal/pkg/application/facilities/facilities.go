package facilities

import (
	"context"
	"sync"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
)

const timeFormat string = "2006-01-02 15:04:05"

type Storage interface {
	StoreBeachesFromSource(context.Context, client.ContextBrokerClient, string, domain.FeatureCollection) error
	StoreSportsFieldsFromSource(context.Context, client.ContextBrokerClient, string, domain.FeatureCollection) error
	StoreSportsVenuesFromSource(context.Context, client.ContextBrokerClient, string, domain.FeatureCollection) error
	StoreTrailsFromSource(context.Context, client.ContextBrokerClient, string, domain.FeatureCollection) error
}

type storageImpl struct {
	deleted map[int64]time.Time
	m       sync.Mutex
}

func NewStorage(ctx context.Context) Storage {
	return &storageImpl{
		deleted: make(map[int64]time.Time),
		m:       sync.Mutex{},
	}
}

func (s *storageImpl) shouldBeDeleted(feature domain.Feature) (bool, bool) {
	if feature.Properties.Published && feature.Properties.Deleted == nil {
		return false, false
	}

	s.m.Lock()
	defer s.m.Unlock()

	if deletedTime, ok := s.deleted[feature.ID]; ok {
		y, m, d := time.Now().UTC().Date()
		midnight := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

		if deletedTime.Before(midnight) {
			delete(s.deleted, feature.ID)
		} else {
			return true, true
		}
	}

	s.deleted[feature.ID] = time.Now().UTC()

	return true, false
}
