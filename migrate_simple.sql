-- Simple migration: Drop and recreate table (WARNING: This will delete all existing data!)
-- Use this only if you don't have important data to preserve

USE teamwork;

-- Drop existing tables
DROP TABLE IF EXISTS projects;

DROP TABLE IF EXISTS users;

-- Create the users table with updated structure
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    tg_id BIGINT NOT NULL UNIQUE,
    tg_name VARCHAR(255),
    email VARCHAR(255),
    name VARCHAR(255),
    ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_tg_id (tg_id)
);

-- Create the projects table
CREATE TABLE projects (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status ENUM(
        'planning',
        'active',
        'paused',
        'completed',
        'cancelled'
    ) DEFAULT 'planning',
    priority ENUM(
        'low',
        'medium',
        'high',
        'urgent'
    ) DEFAULT 'medium',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deadline DATE NULL,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_priority (priority)
);

-- Verify the structure
DESCRIBE users;

DESCRIBE projects;