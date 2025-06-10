-- Create database if not exists
CREATE DATABASE IF NOT EXISTS teamwork;

-- Use the database
USE teamwork;

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    tg_id BIGINT NOT NULL UNIQUE,
    tg_name VARCHAR(255),
    email VARCHAR(255),
    name VARCHAR(255),
    ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_tg_id (tg_id)
);

-- Create projects table
CREATE TABLE IF NOT EXISTS projects (
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

-- Example inserts
-- INSERT INTO users (tg_id, tg_name, email, name) VALUES (12345678, 'username', 'user@example.com', 'User Name');
-- INSERT INTO projects (user_id, title, description, status, priority) VALUES (1, 'My First Project', 'Description of my project', 'active', 'high');