CREATE TABLE IF NOT EXISTS devices (
    id SERIAL PRIMARY KEY,
    device_key VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_poll_at TIMESTAMP
);

CREATE INDEX idx_devices_device_key ON devices(device_key);

CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(255) PRIMARY KEY,
    topic VARCHAR(255) NOT NULL,
    to_number VARCHAR(20) NOT NULL,
    body TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP,
    failed_at TIMESTAMP,
    failure_reason TEXT,
    assigned_device_id INTEGER REFERENCES devices(id) ON DELETE SET NULL
);

CREATE INDEX idx_messages_topic ON messages(topic);
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_to_number ON messages(to_number);
CREATE INDEX idx_messages_created_at ON messages(created_at);
CREATE INDEX idx_messages_assigned_device ON messages(assigned_device_id);
CREATE INDEX idx_messages_topic_status ON messages(topic, status);

CREATE TABLE IF NOT EXISTS device_topics (
    id SERIAL PRIMARY KEY,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    topic VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_id, topic)
);

CREATE INDEX idx_device_topics_device_id ON device_topics(device_id);
CREATE INDEX idx_device_topics_topic ON device_topics(topic);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO schema_migrations (version) VALUES (1);
