-- v0 -> v1: Latest revision

CREATE TABLE bots (
    mxid       TEXT PRIMARY KEY,
    owner_mxid TEXT NOT NULL
) STRICT;
