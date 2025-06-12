-- Migration to remove priority and deadline fields from projects table
-- This migration removes priority and deadline columns and their indexes

USE teamwork;

-- Remove indexes first
DROP INDEX IF EXISTS idx_priority ON projects;

-- Remove priority and deadline columns
ALTER TABLE projects
DROP COLUMN IF EXISTS priority,
DROP COLUMN IF EXISTS deadline;

-- Verify the structure
DESCRIBE projects;