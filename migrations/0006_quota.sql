-- +goose Up
CREATE TABLE quota_ledgers (
    user_id       TEXT NOT NULL,
    accounting_day TEXT NOT NULL,
    budget        INTEGER NOT NULL,
    reserved      INTEGER NOT NULL,
    charged       INTEGER NOT NULL,
    warned_at_80  INTEGER NOT NULL,
    exhausted     INTEGER NOT NULL,
    version       INTEGER NOT NULL,
    updated_at    DATETIME NOT NULL,
    PRIMARY KEY (user_id, accounting_day),
    CHECK (reserved >= 0),
    CHECK (charged >= 0),
    CHECK (reserved + charged <= budget)
);

CREATE TABLE quota_reservations (
    reservation_id  TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    accounting_day  TEXT NOT NULL,
    units           INTEGER NOT NULL,
    call            TEXT NOT NULL,
    state           TEXT NOT NULL,
    created_at      DATETIME NOT NULL,
    expires_at      DATETIME NOT NULL,
    CHECK (units > 0),
    CHECK (state IN ('reserved', 'charged', 'released'))
);

CREATE INDEX quota_reservations_state_expires
    ON quota_reservations(state, expires_at);

-- +goose Down
DROP INDEX quota_reservations_state_expires;
DROP TABLE quota_reservations;
DROP TABLE quota_ledgers;
