package main

import (
	"context"
	"runtime/debug"
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

func backgroundMarkRead(ctx context.Context, evt *event.Event) {
	go func() {
		err := cli.MarkRead(evt.RoomID, evt.ID)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to mark command as read")
		}
	}()
}

func handleCommand(ctx context.Context, evt *event.Event) {
	log := *zerolog.Ctx(ctx)
	defer func() {
		if r := recover(); r != nil {
			logEvt := log.Error()
			if err, ok := r.(error); ok {
				logEvt = logEvt.Err(err)
			} else {
				logEvt = logEvt.Interface("error", r)
			}
			logEvt.Bytes("stack", debug.Stack()).Msg("Panic while processing command")
			reply(ctx, "Internal error processing command.")
		}
	}()

	content, ok := evt.Content.Parsed.(*event.MessageEventContent)
	if !ok {
		log.Debug().Type("content_type", evt.Content).Msg("Ignoring message with unknown parsed content type")
		return
	} else if content.RelatesTo.GetReplaceID() != "" {
		log.Debug().Msg("Ignoring edit event")
		return
	} else if content.MsgType == event.MsgNotice {
		log.Debug().Msg("Ignoring m.notice message")
		return
	}
	content.RemoveReplyFallback()
	args := strings.Fields(content.Body)
	if len(args) == 0 {
		log.Debug().Msg("Ignoring empty message")
		return
	}

	command := strings.TrimPrefix(strings.ToLower(args[0]), "!")
	log = log.With().Str("command", command).Logger()

	cmdCtx := getCommandContextFromMap(evt.Sender)
	ctx = context.WithValue(ctx, contextKeyCmdContext, cmdCtx)
	ctx = log.WithContext(ctx)

	cmdCtx.Lock()
	defer cmdCtx.Unlock()

	if cmdCtx.Next != nil && command != "cancel" {
		backgroundMarkRead(ctx, evt)
		cmdCtx.Next(ctx, args)
	} else if content.MsgType != event.MsgText {
		log.Debug().Msg("Ignoring non-text non-context command")
	} else {
		backgroundMarkRead(ctx, evt)
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
