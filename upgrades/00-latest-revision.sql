-- v0 -> v1: Latest revision

CREATE TABLE bots (
    mxid       TEXT PRIMARY KEY,
    owner_mxid TEXT NOT NULL
) STRICT;

CREATE TABLE self_destructing_events (
    event_id  TEXT PRIMARY KEY,
    room_id   TEXT NOT NULL,
    delete_at BIGINT NOT NULL
) STRICT;
