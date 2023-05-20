package main

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/id"
)

func restartSelfDestruct() {
	log := globalLog.With().Str("action", "restart self-destruct").Logger()
	ctx := log.WithContext(context.Background())
	evts, err := db.GetSelfDestructingEvents(ctx)
	if err != nil {
		log.Err(err).Msg("Failed to get self-destructing events from database")
		return
	}
	for _, evt := range evts {
		log.Debug().
			Str("event_id", evt.EventID.String()).
			Time("delete_at", evt.DeleteAt).
			Msg("Restarting self-destruct timer")
		go doSelfDestruct(ctx, evt.RoomID, evt.EventID, evt.DeleteAt)
	}
}

func selfDestruct(ctx context.Context, eventID id.EventID, after time.Duration) {
	roomID := getEvent(ctx).RoomID
	deleteAt := time.Now().Add(after)
	err := db.SetSelfDestruct(ctx, roomID, eventID, deleteAt)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to mark event as self-destructing in database")
	}
	go doSelfDestruct(ctx, roomID, eventID, deleteAt)
}

func doSelfDestruct(ctx context.Context, roomID id.RoomID, eventID id.EventID, at time.Time) {
	time.Sleep(time.Until(at))
	log := zerolog.Ctx(ctx).With().Str("target_event_id", eventID.String()).Logger()
	if _, err := cli.RedactEvent(roomID, eventID); err != nil {
		log.Err(err).Msg("Failed to self-destruct message")
	} else {
		log.Debug().Msg("Event self-destructed")
		err = db.DoneSelfDestruct(ctx, eventID)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to remove self-destruct marker from database")
		}
	}
}
