package facilities

import (
	"time"

	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
)

const timeFormat string = "2006-01-02 15:04:05"

var deleted map[int64]time.Time = make(map[int64]time.Time)

func shouldBeDeleted(feature domain.Feature) (bool, bool) {
	if feature.Properties.Published && feature.Properties.Deleted == nil {
		return false, false
	}

	if deletedTime, ok := deleted[feature.ID]; ok {
		y, m, d := time.Now().UTC().Date()
		midnight := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

		if deletedTime.Before(midnight) {
			delete(deleted, feature.ID)
		} else {
			return true, true
		}
	}

	deleted[feature.ID] = time.Now().UTC()
	return true, false
}
