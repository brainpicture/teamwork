-- Migration script for improved project management schema
-- Creates projects table without user_id and project_users table for user-project relationships with roles

USE teamwork;

-- Drop existing projects table if it exists (clean start)
DROP TABLE IF EXISTS project_users;

DROP TABLE IF EXISTS projects;

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