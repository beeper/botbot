package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const helpMessage = `Botbot %s

Commands:
* ´ping´: Pings the bot
* ´help´: Shows this message
* ´list´: Show a list of your bots
* ´show <username>´: Show info about a specific bot
* ´create <username>´: Register a new bot
* ´reset <username>´: Reset the access token of a bot
`

type CommandHandler func(ctx context.Context, args []string)

var commands = map[string]CommandHandler{
	"ping":   cmdPing,
	"help":   cmdHelp,
	"list":   cmdList,
	"show":   cmdShow,
	"create": cmdCreate,
	"reset":  cmdReset,
}

func handleCommand(ctx context.Context, evt *event.Event) {
	content := evt.Content.AsMessage()
	args := strings.Fields(content.Body)
	command := strings.TrimPrefix(strings.ToLower(args[0]), "!")
	args = args[1:]
	cmd, ok := commands[command]
	if !ok {
		cmd = cmdUnknownCommand
	}
	cmd(ctx, args)
}

func cmdUnknownCommand(ctx context.Context, _ []string) {
	reply(ctx, "Unknown command. Use `help` for help.")
}

func cmdPing(ctx context.Context, _ []string) {
	reply(ctx, "Pong!")
}

func cmdHelp(ctx context.Context, _ []string) {
	reply(ctx, strings.ReplaceAll(helpMessage, "´", "`"), Version)
}

func cmdList(ctx context.Context, args []string) {
	bots, err := db.GetBots(ctx, getEvent(ctx).Sender)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to get bot list from database")
		reply(ctx, "Failed to get bot list")
		return
	}
	if len(bots) == 0 {
		reply(ctx, "You don't have any bots 😿")
		return
	}
	lines := make([]string, len(bots))
	for i, bot := range bots {
		lines[i] = fmt.Sprintf("* [%s](%s)", bot.MXID, bot.MXID.URI().MatrixToURL())
	}
	reply(ctx, "Your bots:\n\n"+strings.Join(lines, "\n"))
}

func getBotMeta(ctx context.Context, username string) *Bot {
	bot, err := db.GetBot(ctx, id.NewUserID(strings.ToLower(username), cli.UserID.Homeserver()))
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to get bot from database")
		reply(ctx, "Failed to get bot")
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
	reply(ctx, "Showing bot info is not yet implemented")
}

func cmdReset(ctx context.Context, args []string) {
	if len(args) < 1 {
		reply(ctx, "**Usage:** `reset <username>`")
		return
	}
	bot := getBotMeta(ctx, args[0])
	if bot == nil {
		return
	}
	reply(ctx, "Resetting bot devices is not yet implemented")
}
