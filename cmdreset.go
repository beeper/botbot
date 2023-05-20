package main

import (
	"context"

	"maunium.net/go/mautrix/synapseadmin"
	"maunium.net/go/mautrix/util"
)

func cmdReset(ctx context.Context, args []string) {
	if len(args) < 1 {
		reply(ctx, "**Usage:** `reset <username>`")
		return
	}
	bot := getBotMeta(ctx, args[0])
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
