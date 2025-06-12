-- Add tasks table
-- This migration adds support for tasks within projects

USE teamwork;

-- Create tasks table
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