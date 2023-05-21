package main

import (
	"context"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/exp/maps"
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
	"delete": cmdDelete,
	"cancel": cmdCancel,

	// Aliases
	"register":   cmdCreate,
	"info":       cmdShow,
	"get":        cmdShow,
	"remove":     cmdDelete,
	"unregister": cmdDelete,
}

type CommandContext struct {
	sync.Mutex
	Next   CommandHandler
	Action string
	Data   map[string]any
}

func (cmdCtx *CommandContext) Clear() {
	cmdCtx.Next = nil
	cmdCtx.Action = ""
	maps.Clear(cmdCtx.Data)
}

var commandContext = make(map[id.UserID]*CommandContext)
var commandContextLock sync.Mutex

func getCommandContextFromMap(userID id.UserID) *CommandContext {
	commandContextLock.Lock()
	ctx, ok := commandContext[userID]
	if !ok {
		ctx = &CommandContext{
			Data: make(map[string]any),
		}
		commandContext[userID] = ctx
	}
	commandContextLock.Unlock()
	return ctx
}

func getUserCommandContext(ctx context.Context) *CommandContext {
	return ctx.Value(contextKeyCmdContext).(*CommandContext)
}

func handleCommand(ctx context.Context, evt *event.Event) {
	go func() {
		err := cli.MarkRead(evt.RoomID, evt.ID)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to mark command as read")
		}
	}()

	content := evt.Content.AsMessage()
	args := strings.Fields(content.Body)
	cmdCtx := getCommandContextFromMap(evt.Sender)
	ctx = context.WithValue(ctx, contextKeyCmdContext, cmdCtx)
	cmdCtx.Lock()
	defer cmdCtx.Unlock()

	command := strings.TrimPrefix(strings.ToLower(args[0]), "!")
	if cmdCtx.Next != nil && command != "cancel" {
		cmdCtx.Next(ctx, args)
	} else {
		args = args[1:]
		cmd, ok := commands[command]
		if !ok {
			cmd = cmdUnknownCommand
		}
		cmd(ctx, args)
	}
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

func cmdCancel(ctx context.Context, _ []string) {
	cmdCtx := getUserCommandContext(ctx)
	if cmdCtx.Action == "" && cmdCtx.Next == nil {
		reply(ctx, "Nothing to cancel")
	} else {
		reply(ctx, "Cancelled %s", cmdCtx.Action)
		cmdCtx.Clear()
	}
}
