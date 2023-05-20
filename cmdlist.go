package main

import (
	"context"
	"fmt"
	"strings"
)

func cmdList(ctx context.Context, args []string) {
	bots, err := db.GetBots(ctx, getEvent(ctx).Sender)
	if err != nil {
		replyErr(ctx, err, "Failed to get bot list")
	} else if len(bots) == 0 {
		reply(ctx, "You don't have any bots 😿")
	} else {
		lines := make([]string, len(bots))
		for i, bot := range bots {
			lines[i] = fmt.Sprintf("* [%s](%s)", bot.MXID, bot.MXID.URI().MatrixToURL())
		}
		reply(ctx, "Your bots:\n\n"+strings.Join(lines, "\n"))
	}
}
