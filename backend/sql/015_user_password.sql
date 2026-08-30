-- Migration 015: Add password_hash column to users table for direct email/password auth
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;
