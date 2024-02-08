package facilities

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/matryer/is"
)

func TestShouldBeDeleted(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	var aWeekAgo = time.Now().UTC().Add(-1 * 7 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	props := domain.FeatureProps{
		Deleted: &aWeekAgo,
	}

	storage := NewStorage(ctx)
	impl := storage.(*storageImpl)

	f := domain.Feature{ID: 1, Properties: props}

	ok, alreadyDeleted := impl.shouldBeDeleted(ctx, f)

	is.True(ok)
	is.True(!alreadyDeleted)

	impl.deleted[1] = time.Now().UTC()

	ok, alreadyDeleted = impl.shouldBeDeleted(ctx, f)

	is.True(ok)
	is.True(alreadyDeleted)
}
