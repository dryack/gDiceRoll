-- 001_create_hello_messages.up.sql
CREATE TABLE IF NOT EXISTS hello_messages (
                                              key TEXT PRIMARY KEY,
                                              message TEXT NOT NULL
);