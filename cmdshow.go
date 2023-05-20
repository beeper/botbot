package main

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/id"

	"maunium.net/go/mautrix/util"
)

const showBotMessage = `Bot ´%s´ info:

* Created on %s
* Device ID: ´%s´
* Last seen %s
`

func getBotMeta(ctx context.Context, username string) *Bot {
	bot, err := db.GetBot(ctx, id.NewUserID(strings.ToLower(username), cli.UserID.Homeserver()))
	if err != nil {
		replyErr(ctx, err, "Failed to get bot info")
	} else if bot == nil {
		reply(ctx, "That bot doesn't exist")
	} else if bot.OwnerMXID != getEvent(ctx).Sender {
		reply(ctx, "That's not your bot")
	} else {
		return bot
	}
	return nil
}

func cmdShow(ctx context.Context, args []string) {
	if len(args) < 1 {
		reply(ctx, "**Usage:** `show <username>`")
		return
	}
	bot := getBotMeta(ctx, args[0])
	if bot == nil {
		return
	}
	userInfo, err := synadm.GetUserInfo(ctx, bot.MXID)
	if err != nil {
		replyErr(ctx, err, "Failed to get bot user info")
		return
	}
	devices, err := synadm.ListDevices(ctx, bot.MXID)
	if err != nil {
		replyErr(ctx, err, "Failed to get bot device info")
		return
	}
	if len(devices.Devices) > 1 {
		zerolog.Ctx(ctx).Warn().Msg("Bot has multiple devices")
	}
	var deviceID, lastSeen string
	if len(devices.Devices) > 0 {
		deviceInfo := devices.Devices[0]
		deviceID = deviceInfo.DeviceID.String()

		lastSeenTS := time.UnixMilli(deviceInfo.LastSeenTS)
		if deviceInfo.LastSeenTS == 0 {
			lastSeen = "never"
		} else if seenAgo := time.Since(lastSeenTS); seenAgo < time.Second {
			lastSeen = "now"
		} else if seenAgo >= 1*util.Week {
			lastSeen = "at " + lastSeenTS.UTC().Format(time.UnixDate)
		} else {
			lastSeen = util.FormatDuration(seenAgo) + " ago"
		}
		if deviceInfo.LastSeenIP != "" {
			lastSeen += " from " + deviceInfo.LastSeenIP
		}
	} else {
		zerolog.Ctx(ctx).Warn().Msg("Bot has no devices")
		deviceID = "<none>"
		lastSeen = "N/A"
	}
	reply(
		ctx, showBotMessage,
		userInfo.UserID,
		userInfo.CreationTS.UTC().Format(time.UnixDate),
		deviceID,
		lastSeen,
	)
}
