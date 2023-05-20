package main

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util/dbutil"
)

type Database struct {
	*dbutil.Database
}

type Bot struct {
	MXID      id.UserID
	OwnerMXID id.UserID
}

const (
	registerBot    = "INSERT INTO bots (mxid, owner_mxid) VALUES ($1, $2)"
	getBotsByOwner = "SELECT mxid, owner_mxid FROM bots WHERE owner_mxid=$1"
	getBot         = "SELECT mxid, owner_mxid FROM bots WHERE mxid=$1"
)

func (db *Database) RegisterBot(ctx context.Context, owner, bot id.UserID) error {
	_, err := db.ExecContext(ctx, registerBot, bot, owner)
	return err
}

func (db *Database) GetBots(ctx context.Context, owner id.UserID) ([]Bot, error) {
	rows, err := db.QueryContext(ctx, getBotsByOwner, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bots []Bot
	for rows.Next() {
		var bot Bot
		if err = rows.Scan(&bot.MXID, &bot.OwnerMXID); err != nil {
			return nil, err
		}
		bots = append(bots, bot)
	}
	return bots, rows.Err()
}

func (db *Database) GetBot(ctx context.Context, bot id.UserID) (*Bot, error) {
	var b Bot
	err := db.
		QueryRowContext(ctx, getBot, bot).
		Scan(&b.MXID, &b.OwnerMXID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &b, err
}

const (
	setSelfDestruct    = "INSERT INTO self_destructing_events (event_id, room_id, delete_at) VALUES ($1, $2, $3)"
	getSelfDestruct    = "SELECT event_id, room_id, delete_at FROM self_destructing_events"
	deleteSelfDestruct = "DELETE FROM self_destructing_events WHERE event_id=$1"
)

func (db *Database) SetSelfDestruct(ctx context.Context, roomID id.RoomID, eventID id.EventID, deleteAt time.Time) error {
	_, err := db.ExecContext(ctx, setSelfDestruct, eventID, roomID, deleteAt.UnixMilli())
	return err
}

type SelfDestructingEvent struct {
	EventID  id.EventID
	RoomID   id.RoomID
	DeleteAt time.Time
}

func (db *Database) GetSelfDestructingEvents(ctx context.Context) ([]SelfDestructingEvent, error) {
	rows, err := db.QueryContext(ctx, getSelfDestruct)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []SelfDestructingEvent
	for rows.Next() {
		var evt SelfDestructingEvent
		var deleteTs int64
		if err = rows.Scan(&evt.EventID, &evt.RoomID, &deleteTs); err != nil {
			return nil, err
		}
		evt.DeleteAt = time.UnixMilli(deleteTs)
		events = append(events, evt)
	}
	return events, rows.Err()
}

func (db *Database) DoneSelfDestruct(ctx context.Context, eventID id.EventID) error {
	_, err := db.ExecContext(ctx, deleteSelfDestruct, eventID)
	return err
}
