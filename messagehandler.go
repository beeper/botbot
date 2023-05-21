package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

type contextKey int

const (
	contextKeyEvent contextKey = iota
	contextKeyCmdContext
)

func getEvent(ctx context.Context) *event.Event {
	evt, ok := ctx.Value(contextKeyEvent).(*event.Event)
	if !ok {
		panic("tried to get event from context that doesn't have event")
	}
	return evt
}

func replyErr(ctx context.Context, err error, message string) {
	zerolog.Ctx(ctx).Err(err).Msg(message)
	reply(ctx, message)
}

func reply(ctx context.Context, message string, args ...any) id.EventID {
	evt := getEvent(ctx)
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	message = strings.ReplaceAll(message, "´", "`")
	content := format.RenderMarkdown(message, true, true)
	content.MsgType = event.MsgNotice
	resp, err := cli.SendMessageEvent(evt.RoomID, event.EventMessage, content)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to send reply")
		return ""
	}
	zerolog.Ctx(ctx).Debug().
		Str("reply_event_id", resp.EventID.String()).
		Msg("Sent reply")
	return resp.EventID
}

func handleMessage(source mautrix.EventSource, evt *event.Event) {
	if evt.Sender == cli.UserID {
		return
	}
	log := globalLog.With().
		Str("event_id", evt.ID.String()).
		Str("action", "incoming message").
		Logger()
	log.Debug().
		Str("room_id", evt.RoomID.String()).
		Str("sender", evt.Sender.String()).
		Time("message_timestamp", time.UnixMilli(evt.Timestamp)).
		Msg("Received message event")
	ctx := log.WithContext(context.WithValue(context.Background(), contextKeyEvent, evt))
	if expectedUserID, err := getOtherUserID(ctx, evt.RoomID, true, true); err != nil {
		log.Warn().Err(err).Msg("Ignoring message: failed to check expected user ID in room")
	} else if expectedUserID != evt.Sender {
		log.Debug().Str("expected_sender", expectedUserID.String()).Msg("Ignoring message from unexpected user")
	} else if time.Since(time.UnixMilli(evt.Timestamp)) > 5*time.Minute {
		log.Debug().Msg("Ignoring message older than 5 minutes")
	} else if !evt.Mautrix.WasEncrypted {
		log.Debug().Msg("Dropping unencrypted message")
		reply(ctx, "This bot only responds to encrypted messages")
	} else if evt.Mautrix.TrustState < id.TrustStateCrossSignedTOFU {
		log.Debug().
			Str("trust_state", evt.Mautrix.TrustState.String()).
			Bool("forwarded_keys", evt.Mautrix.ForwardedKeys).
			Msg("Dropping message with insufficient verification level")
		reply(ctx, "Insufficient verification level %s (forwarded keys: %t)", evt.Mautrix.TrustState, evt.Mautrix.ForwardedKeys)
	} else {
		handleCommand(ctx, evt)
	}
}
