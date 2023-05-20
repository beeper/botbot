# botbot
The bot that manages bot accounts on beeper.com. Should work on a normal Synapse instance too.

## Configuration
The bot is configured through environment variables

* `BOTBOT_HOMESERVER_URL` - The Matrix homeserver URL.
  Can (and probably should) be local.
* `BOTBOT_USERNAME` - The username for this bot.
  * The user must be a Synapse admin for resetting bot passwords and viewing
    bot user info.
* `BOTBOT_PASSWORD` - The password for the user above.
* `BOTBOT_DATABASE_URI` - The database URI (file name for SQLite or full
  connection string for Postgres). Defaults to `botbot.db`.
* `BOTBOT_DATABASE_TYPE` - The type of database. `postgres` for postgres,
  `sqlite3-fk-wal` for SQLite.
* `BOTBOT_PICKLE_KEY` - Pickle key for encrypting encryption keys.
* `BOTBOT_REGISTER_SECRET` - Registration shared secret for creating new bot
  accounts for users. Required unless the Beeper API URL is set.
* `BOTBOT_BEEPER_API_URL` - Optional Beeper API server URL for registering
  users through the Beeper API instead of directly with Synapse.
* `BOTBOT_LOG_LEVEL` - Log level. Defaults to `debug`.
* `BOTBOT_MAX_BOTS_PER_USER` - Maximum number of bots that a single user can
  create. Defaults to 10. Limit is disabled if set to 0.

## Docker image
The docker image built by GitHub actions is available in the GitHub registry:
[`ghcr.io/beeper/botbot`](https://github.com/beeper/botbot/pkgs/container/botbot)

You can use `:latest` for the latest commit, or a git commit hash to pin to a
specific version. Only amd64 images are currently available.

## Usage notes
The bot will only accept encrypted DM invitations and will leave if any other
users join the room. After creating a DM, use `help` for help. Prefixing
commands is not necessary as there won't be any other bots in the room.

The bot enforces cross-signing (with trust-on-first-use for the master key)
and will reject messages from unverified devices. Currently, the only way to
reset TOFU is to manually change the `first_seen_key` column in the
`crypto_cross_signing_keys` table in the database.
