package main

import (
	"context"
	"database/sql"
	"errors"

	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util/dbutil"
)

type Database struct {
	*dbutil.Database
}

type Bot struct {
	MXID      id.UserID
	OwnerMXID id.UserID
	Password  string
}

func (db *Database) RegisterBot(ctx context.Context, owner, bot id.UserID, password string) error {
	_, err := db.ExecContext(ctx, "INSERT INTO bots (mxid, owner_mxid, password) VALUES ($1, $2, $3)", bot, owner, password)
	return err
}

func (db *Database) GetBots(ctx context.Context, owner id.UserID) ([]Bot, error) {
	rows, err := db.QueryContext(ctx, "SELECT mxid, owner_mxid, password FROM bots WHERE owner_mxid=$1", owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bots []Bot
	for rows.Next() {
		var bot Bot
		if err = rows.Scan(&bot.MXID, &bot.OwnerMXID, &bot.Password); err != nil {
			return nil, err
		}
		bots = append(bots, bot)
	}
	return bots, rows.Err()
}

func (db *Database) GetBot(ctx context.Context, bot id.UserID) (*Bot, error) {
	var b Bot
	err := db.QueryRowContext(ctx, "SELECT mxid, owner_mxid, password FROM bots WHERE mxid=$1", bot).Scan(&b.MXID, &b.OwnerMXID, &b.Password)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &b, err
}

func (db *Database) GetCommandContext(ctx context.Context, userID id.UserID, roomID id.RoomID) (map[string]any, error) {
	return nil, nil
}
