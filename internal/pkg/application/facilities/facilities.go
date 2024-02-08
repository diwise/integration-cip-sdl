package facilities

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
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

// shouldBeDeleted maintains a cache of deleted features so that we do not
// call delete on the same entity for every update
func (s *storageImpl) shouldBeDeleted(ctx context.Context, feature domain.Feature) (okToDelete bool, alreadyDeleted bool) {
	if feature.Properties.Published && feature.Properties.Deleted == nil {
		return
	}

	s.m.Lock()
	defer s.m.Unlock()

	deleteWindowDuration := 30 * 24 * time.Hour // ignore deletions from more than 30 days ago
	startOfDeleteWindow := time.Now().UTC().Add(-1 * deleteWindowDuration)

	// check if this feature has already been marked for deletion
	if deletedTime, ok := s.deleted[feature.ID]; ok {

		// Forget deleted features that fall outside of the delete window to prevent
		// the cache from growing too large
		if deletedTime.Before(startOfDeleteWindow) {
			delete(s.deleted, feature.ID)
		}

		okToDelete = true
		alreadyDeleted = true
		return
	}

	// returns the first non nil decodeable timestamp
	findMostRecentTimestamp := func(timestamps ...*string) time.Time {
		const timeFormat string = "2006-01-02 15:04:05"
		for _, ts := range timestamps {
			if ts != nil {
				t, err := time.Parse(timeFormat, *ts)
				if err != nil {
					continue
				}
				return t
			}
		}

		return time.Time{}
	}

	deletedAt := findMostRecentTimestamp(feature.Properties.Deleted, feature.Properties.Updated, feature.Properties.Created)

	if deletedAt.Before(startOfDeleteWindow) {
		logging.GetFromContext(ctx).Info("feature was deleted/unpublished a long time ago", slog.Int64("featureID", feature.ID), "when", deletedAt.Format(time.RFC3339Nano))
		okToDelete = true
		alreadyDeleted = true
	} else {
		logging.GetFromContext(ctx).Info("feature was recently deleted/unpublished", slog.Int64("featureID", feature.ID), "when", deletedAt.Format(time.RFC3339Nano))
		s.deleted[feature.ID] = deletedAt
		okToDelete = true
	}

	return
}
