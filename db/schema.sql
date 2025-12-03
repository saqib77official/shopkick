-- SQLite schema for suggestions feature
-- Run: sqlite3 suggestions.db < db/schema.sql

PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS suggestions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    suggestion TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_suggestions_created_at ON suggestions (created_at DESC);
