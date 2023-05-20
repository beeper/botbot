package main

import (
	"context"

	"maunium.net/go/mautrix/id"
)

const usernameInvalidError = `That username is not valid. Usernames must:

* Be between 5 and 32 characters long in total (i.e. 2-29 characters + bot suffix)
* Only contain lowercase letters (a-z), numbers (0-9) and dashes (-)
* Not start with dash
* End with ´bot´`

const botDetails = `

* User ID: ´%s´
* Device ID: ´%s´
* Access token: ´%s´`

func cmdCreate(ctx context.Context, args []string) {
	if len(args) < 1 {
		reply(ctx, "**Usage:** `create <username>`")
		return
	}
	username := args[0]
	userID := id.NewUserID(username, cli.UserID.Homeserver())
	if !IsValidBotUsername(username) {
		reply(ctx, usernameInvalidError)
	} else if existingBot, err := db.GetBot(ctx, userID); err != nil {
		replyErr(ctx, err, "Failed to check if bot already exists in database")
	} else if existingBot != nil {
		if existingBot.OwnerMXID == getEvent(ctx).Sender {
			reply(ctx, "You've already registered that bot. You can use `reset <username>` to reset the token.")
		} else {
			reply(ctx, "That username is already taken")
		}
	} else if available, err := IsUsernameAvailable(ctx, username); err != nil {
		replyErr(ctx, err, "Failed to check username availability")
	} else if !available {
		reply(ctx, "That username is already taken")
	} else if password, err := RegisterUser(ctx, username); err != nil {
		replyErr(ctx, err, "Failed to register bot")
	} else if err = db.RegisterBot(ctx, getEvent(ctx).Sender, userID); err != nil {
		replyErr(ctx, err, "Failed to store registered bot in database")
	} else if device, err := Login(ctx, userID, password); err != nil {
		replyErr(ctx, err, "Failed to log in as bot after registering")
	} else {
		reply(ctx, "Bot created successfully 🎉"+botDetails, device.UserID, device.DeviceID, device.AccessToken)
	}
}
