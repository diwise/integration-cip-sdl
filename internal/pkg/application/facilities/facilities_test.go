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

	var d string = "2022-01-01 00:00:00"
	props := domain.FeatureProps{
		Deleted: &d,
	}

	storage := NewStorage(context.Background())
	impl := storage.(*storageImpl)

	f := domain.Feature{ID: 1, Properties: props}

	impl.deleted[1] = time.Now().UTC()

	ok, alreadyDeleted := impl.shouldBeDeleted(f)

	is.True(ok)
	is.True(alreadyDeleted)

	impl.deleted[1] = time.Now().UTC().Add(-1 * 25 * time.Hour)

	ok, alreadyDeleted = impl.shouldBeDeleted(f)

	is.True(ok)
	is.True(!alreadyDeleted)
}
