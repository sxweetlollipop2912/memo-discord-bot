CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(50) PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    discord_channel_id VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS memos (
    id SERIAL PRIMARY KEY,
    discord_user_id VARCHAR(50) NOT NULL,
    discord_channel_id VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    remind_at TIMESTAMP WITH TIME ZONE NOT NULL,
    sent BOOLEAN DEFAULT FALSE,
    CONSTRAINT remind_at_check CHECK (remind_at > created_at)
); 