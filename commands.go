package main

import (
	"context"
	"strings"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/event"
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
	"delete": cmdDelete,

	// Aliases
	"register":   cmdCreate,
	"info":       cmdShow,
	"get":        cmdShow,
	"remove":     cmdDelete,
	"unregister": cmdDelete,
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
	go func() {
		err := cli.MarkRead(evt.RoomID, evt.ID)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to mark command as read")
		}
	}()
	cmd(ctx, args)
}

func cmdUnknownCommand(ctx context.Context, _ []string) {
	reply(ctx, "Unknown command. Use `help` for help.")
}

func cmdPing(ctx context.Context, _ []string) {
	reply(ctx, "Pong!")
}

func cmdHelp(ctx context.Context, _ []string) {
	reply(ctx, helpMessage, Version)
}
