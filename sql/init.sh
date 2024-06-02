#!/bin/bash

set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
CREATE TABLE IF NOT EXISTS urls (
    short_url VARCHAR(255) PRIMARY KEY,
    original_url TEXT NOT NULL,
    expiry TIMESTAMP NOT NULL,
    click_count INT DEFAULT 0
);
EOSQL
