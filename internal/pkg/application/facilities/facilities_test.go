package facilities

import (
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

	f := domain.Feature{ID: 1, Properties: props}

	deleted[1] = time.Now().UTC()

	ok, alreadyDeleted := shouldBeDeleted(f)

	is.True(ok)
	is.True(alreadyDeleted)

	deleted[1] = time.Now().UTC().Add(-1 * 25 * time.Hour)

	ok, alreadyDeleted = shouldBeDeleted(f)

	is.True(ok)
	is.True(!alreadyDeleted)	
}
