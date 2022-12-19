package facilities

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/integration-cip-sdl/internal/pkg/domain"
	"github.com/rs/zerolog"
)

func deleteEntity(ctx context.Context, ctxBrokerClient client.ContextBrokerClient, logger zerolog.Logger, feature domain.Feature) {
	var timeFormat string = "2006-01-02 15:04:05"
	if t, err := time.Parse(timeFormat, *feature.Properties.Deleted); err == nil {
		oneWeekAgo := time.Now().UTC().Add(-1 * (24*7*time.Hour))
		if t.Before(oneWeekAgo) {
			return
		}
	}

	entityID := fmt.Sprintf("%s%d", domain.SundsvallAnlaggningPrefix, feature.ID)
	_, err := ctxBrokerClient.DeleteEntity(ctx, entityID)
	if err != nil {
		logger.Info().Msgf("delete entity %s failed with error \"%s\"", entityID, err.Error())
	}
}