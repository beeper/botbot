-- v0 -> v1: Latest revision

CREATE TABLE bots (
    mxid       TEXT NOT NULL PRIMARY KEY,
    owner_mxid TEXT NOT NULL
);

CREATE TABLE self_destructing_events (
    event_id  TEXT   NOT NULL PRIMARY KEY,
    room_id   TEXT   NOT NULL,
    delete_at BIGINT NOT NULL
);
