-- Add current_project_id field to users table
-- This allows tracking which project the user is currently working on

USE teamwork;

-- Add current_project_id field to users table
ALTER TABLE users
ADD COLUMN current_project_id INT NULL,
ADD FOREIGN KEY (current_project_id) REFERENCES projects (id) ON DELETE SET NULL,
ADD INDEX idx_current_project_id (current_project_id);