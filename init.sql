-- Complete database schema initialization
-- Creates database and all tables with improved project management structure

-- Create database
CREATE DATABASE IF NOT EXISTS teamwork;

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

-- Create projects table (without user_id - projects can have multiple users)
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

-- Create project_users table for many-to-many relationship with roles
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

-- Create messages table for conversation context
CREATE TABLE IF NOT EXISTS messages (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    chat_id BIGINT NOT NULL,
    role ENUM('user', 'assistant') NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_chat_created (user_id, chat_id, created_at),
    INDEX idx_chat_created (chat_id, created_at),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);