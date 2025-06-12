-- Simple migration that recreates all tables (WARNING: destroys all data!)
-- This is used for development reset

USE teamwork;

-- Disable foreign key checks to avoid constraint issues
SET FOREIGN_KEY_CHECKS = 0;

-- Drop all tables in correct order (to avoid foreign key constraints)
DROP TABLE IF EXISTS tasks;

DROP TABLE IF EXISTS messages;

DROP TABLE IF EXISTS project_users;

DROP TABLE IF EXISTS projects;

DROP TABLE IF EXISTS users;

-- Re-enable foreign key checks
SET FOREIGN_KEY_CHECKS = 1;

-- Recreate users table
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    tg_id BIGINT NOT NULL UNIQUE,
    tg_name VARCHAR(255),
    email VARCHAR(255),
    name VARCHAR(255),
    current_project_id INT NULL,
    ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_tg_id (tg_id),
    INDEX idx_current_project_id (current_project_id)
);

-- Recreate projects table
CREATE TABLE projects (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status ENUM(
        'planning',
        'active',
        'paused',
        'completed',
        'cancelled'
    ) DEFAULT 'planning',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
);

-- Recreate project_users table
CREATE TABLE project_users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    project_id INT NOT NULL,
    user_id INT NOT NULL,
    role ENUM(
        'owner',
        'admin',
        'member',
        'viewer'
    ) DEFAULT 'member',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    UNIQUE KEY unique_project_user (project_id, user_id),
    INDEX idx_project_id (project_id),
    INDEX idx_user_id (user_id),
    INDEX idx_role (role)
);

-- Recreate messages table
CREATE TABLE messages (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    chat_id BIGINT NOT NULL,
    role ENUM('user', 'assistant') NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    INDEX idx_chat_id (chat_id),
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at),
    INDEX idx_chat_created (chat_id, created_at)
);

-- Recreate tasks table
CREATE TABLE tasks (
    id INT AUTO_INCREMENT PRIMARY KEY,
    project_id INT NOT NULL,
    user_id INT NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status ENUM(
        'todo',
        'in_progress',
        'review',
        'done',
        'cancelled'
    ) DEFAULT 'todo',
    priority ENUM(
        'low',
        'medium',
        'high',
        'urgent'
    ) DEFAULT 'medium',
    deadline DATETIME NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    INDEX idx_project_id (project_id),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_priority (priority),
    INDEX idx_deadline (deadline),
    INDEX idx_created_at (created_at),
    INDEX idx_project_status (project_id, status),
    INDEX idx_user_status (user_id, status)
);

-- Add foreign key constraints that reference other tables
ALTER TABLE users
ADD FOREIGN KEY (current_project_id) REFERENCES projects (id) ON DELETE SET NULL;