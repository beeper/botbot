package main

import (
	"context"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util"
)

func cmdCreate(ctx context.Context, args []string) {
	if len(args) < 1 {
		reply(ctx, "**Usage:** `create <username>`")
		return
	}
	log := zerolog.Ctx(ctx)
	username := args[0]
	userID := id.NewUserID(username, cli.UserID.Homeserver())
	existingBot, err := db.GetBot(ctx, userID)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to get bot from database")
		reply(ctx, "Failed to get bot")
		return
	} else if existingBot != nil {
		if existingBot.OwnerMXID == getEvent(ctx).Sender {
			reply(ctx, "You've already registered that bot. You can use `reset <username>` to reset the token.")
		} else {
			reply(ctx, "That username is already taken")
		}
		return
	}

	if !IsValidBotUsername(username) {
		reply(ctx, `That username is not valid. Usernames must:

* Be between 5 and 32 characters long in total (i.e. 2-29 characters + bot suffix)
* Only contain lowercase letters (a-z), numbers (0-9) and dashes (-)
* Not start with dash
* End with `+"`bot`")
		return
	} else if available, err := IsUsernameAvailable(ctx, username); err != nil {
		log.Err(err).Msg("Failed to check username availability")
		reply(ctx, "Failed to check username availability")
		return
	} else if !available {
		reply(ctx, "That username is already taken")
		return
	}
	password := util.RandomString(32)
	if err = RegisterUser(ctx, username, password); err != nil {
		log.Err(err).Msg("Failed to register bot")
		reply(ctx, "Failed to register bot")
	} else if err = db.RegisterBot(ctx, getEvent(ctx).Sender, userID, password); err != nil {
		log.Err(err).Msg("Failed to store registered bot in database")
		reply(ctx, "Failed to store registered bot in database")
	} else if device, err := Login(ctx, username, password); err != nil {
		log.Err(err).Msg("Failed to log in as bot after registering")
		reply(ctx, "Failed to log in as bot after registering")
	} else {
		reply(ctx, "Bot created successfully 🎉\n\n* User ID: `%s`\n* Device ID: `%s`\n* Access token: `%s`",
			device.UserID, device.DeviceID, device.AccessToken)
	}
}
