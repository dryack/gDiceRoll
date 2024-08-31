-- 00-init-db-and-user.sql

-- Create hello_messages table
CREATE TABLE IF NOT EXISTS hello_messages (
                                              key TEXT PRIMARY KEY,
                                              message TEXT NOT NULL
);

-- Grant all privileges on hello_messages table to the current user
GRANT ALL PRIVILEGES ON TABLE hello_messages TO CURRENT_USER;