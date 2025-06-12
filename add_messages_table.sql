-- Migration to add messages table for conversation context
-- Usage: mysql -u root -p < add_messages_table.sql

USE teamwork;

-- Create messages table to store conversation history
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

-- Show table structure
DESCRIBE messages;

-- Show success message
SELECT 'Messages table created successfully!' as result;