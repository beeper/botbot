package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/caarlos0/env/v8"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/synapseadmin"
	"maunium.net/go/mautrix/util"
	"maunium.net/go/mautrix/util/dbutil"
	_ "maunium.net/go/mautrix/util/dbutil/litestream"

	"github.com/beeper/botbot/upgrades"
)

var (
	Version = "v0.1.0"

	Tag       = ""
	Commit    = ""
	BuildTime = ""
)

type Config struct {
	HomeserverURL string `env:"HOMESERVER_URL,notEmpty"`
	Username      string `env:"USERNAME,notEmpty"`
	Password      string `env:"PASSWORD,notEmpty"`
	DatabasePath  string `env:"DATABASE_PATH" envDefault:"botbot.db"`
	DatabaseType  string `env:"DATABASE_TYPE" envDefault:"sqlite3-fk-wal"`
	PickleKey     string `env:"PICKLE_KEY" envDefault:"meow"`

	BeeperAPIURL string `env:"BEEPER_API_URL"`

	LoginJWTKey    string `env:"LOGIN_JWT_KEY"`
	RegisterSecret string `env:"REGISTER_SECRET"`

	LogLevel zerolog.Level `env:"LOG_LEVEL" envDefault:"debug"`
}

var cli *mautrix.Client
var synadm *synapseadmin.Client
var db *Database
var cfg Config
var globalLog = zerolog.New(os.Stdout).With().Timestamp().Logger()

func main() {
	if Tag != Version {
		if Commit != "" {
			Version += "+dev." + Commit[:8]
		} else {
			Version += "+dev.unknown"
		}
	}
	mautrix.DefaultUserAgent = fmt.Sprintf("botbot/%s %s", Version, mautrix.DefaultUserAgent)

	log := globalLog
	err := env.ParseWithOptions(&cfg, env.Options{Prefix: "BOTBOT_"})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse environment variables")
	}
	log = log.Level(cfg.LogLevel)
	globalLog = log
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.CallerMarshalFunc = util.CallerWithFunctionName
	defaultCtxLog := log.With().Bool("default_context_log", true).Caller().Logger()
	zerolog.DefaultContextLogger = &defaultCtxLog
	log.Info().
		Str("version", Version).
		Str("built_at", BuildTime).
		Str("mautrix_version", mautrix.VersionWithCommit).
		Msg("Initializing botbot")

	cli, err = mautrix.NewClient(cfg.HomeserverURL, "", "")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize mautrix client")
	}
	cli.Log = log
	synadm = &synapseadmin.Client{Client: cli}

	log.Debug().Msg("Initializing database")
	rawDB, err := dbutil.NewWithDialect(cfg.DatabasePath, cfg.DatabaseType)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	rawDB.Log = dbutil.ZeroLogger(log.With().Str("db_section", "main").Logger())
	rawDB.Owner = "botbot"
	rawDB.UpgradeTable = upgrades.Table
	err = rawDB.Upgrade()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to upgrade database")
	}
	db = &Database{Database: rawDB}

	log.Debug().Msg("Initializing crypto helper")
	cryptoHelper, err := cryptohelper.NewCryptoHelper(cli, []byte(cfg.PickleKey), rawDB)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create crypto helper")
	}

	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:       mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: cfg.Username},
		Password:   cfg.Password,
	}
	cryptoHelper.DBAccountID = ""
	err = cryptoHelper.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize crypto helper")
	}
	cli.Crypto = cryptoHelper

	log.Info().Msg("Initialization complete")

	syncer := cli.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.StateMember, handleMember)
	syncer.OnEventType(event.EventMessage, handleMessage)
	syncer.OnSync(cli.MoveInviteState)
	cryptoHelper.DecryptErrorCallback = func(evt *event.Event, err error) {
		_, _ = cli.SendMessageEvent(evt.RoomID, event.EventMessage, &event.MessageEventContent{
			MsgType:   event.MsgNotice,
			Body:      "Failed to decrypt message",
			RelatesTo: (&event.RelatesTo{}).SetReplyTo(evt.ID),
		}, mautrix.ReqSendEvent{DontEncrypt: true})
	}

	syncCtx, cancelSync := context.WithCancel(context.Background())
	var syncStopWait sync.WaitGroup
	syncStopWait.Add(1)

	go func() {
		defer syncStopWait.Done()
		log.Debug().Msg("Starting syncing")
		err = cli.SyncWithContext(syncCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.WithLevel(zerolog.FatalLevel).Err(err).Msg("Fatal error in syncer")
			cancelSync()
		}
	}()
	go restartSelfDestruct()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	var exitCode int
	select {
	case <-c:
		log.Info().Msg("Interrupt received, stopping...")
	case <-syncCtx.Done():
		log.Info().Msg("Syncer stopped, stopping...")
		exitCode = 2
	}

	cancelSync()
	syncStopWait.Wait()
	err = cryptoHelper.Close()
	if err != nil {
		log.Error().Err(err).Msg("Error closing database")
	}
	os.Exit(exitCode)
}
