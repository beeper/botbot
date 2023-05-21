package main

import (
	"context"
	"fmt"
	"strings"

	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/synapseadmin"
	"maunium.net/go/mautrix/util"
)

const resetConfirm = `Are you sure you want to reset the access token of ´%s´?

This will invalidate the existing token and e2ee device keys.
To keep encryption working on the bot, make sure to clear the bot database too.

Type ´really reset´ to confirm deletion.`

func cmdReset(ctx context.Context, args []string) {
	if len(args) < 1 {
		reply(ctx, "**Usage:** `reset <username>`")
		return
	}
	bot := getBotMeta(ctx, args[0])
	if bot == nil {
		return
	}
	cmdCtx := getUserCommandContext(ctx)
	cmdCtx.Next = cmdReallyReset
	cmdCtx.Data["reset_bot_mxid"] = bot.MXID
	cmdCtx.Action = fmt.Sprintf("resetting `%s`", bot.MXID)
	reply(ctx, resetConfirm, bot.MXID)
}

func cmdReallyReset(ctx context.Context, _ []string) {
	cmdCtx := getUserCommandContext(ctx)
	userID := cmdCtx.Data["reset_bot_mxid"].(id.UserID)
	cmdCtx.Clear()
	if strings.TrimSpace(getEvent(ctx).Content.AsMessage().Body) != "really reset" {
		reply(ctx, "Cancelled resetting `%s`", userID)
		return
	}
	bot := getBotMeta(ctx, userID.Localpart())
	if bot == nil {
		return
	}
	password := util.RandomString(72)
	err := synadm.ResetPassword(ctx, synapseadmin.ReqResetPassword{
		UserID:        bot.MXID,
		NewPassword:   password,
		LogoutDevices: true,
	})
	if err != nil {
		replyErr(ctx, err, "Failed to reset bot")
		return
	}
	resp, err := Login(ctx, bot.MXID, password)
	if err != nil {
		replyErr(ctx, err, "Failed to create device after resetting bot")
		return
	}
	evtID := reply(ctx, "Bot reset successfully."+botDetails, resp.UserID, resp.DeviceID, resp.AccessToken)
	selfDestruct(ctx, evtID, botDetailsSelfDestruct)
}
