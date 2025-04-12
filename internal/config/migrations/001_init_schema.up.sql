CREATE TABLE users (
    username VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE rooms (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_by VARCHAR(255) NOT NULL REFERENCES users(username),
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE room_members (
    room_id INTEGER REFERENCES rooms(id),
    username VARCHAR(255) REFERENCES users(username),
    joined_at TIMESTAMP NOT NULL,
    PRIMARY KEY (room_id, username)
);

CREATE TABLE messages (
    id SERIAL PRIMARY KEY,
    from_user VARCHAR(255) NOT NULL REFERENCES users(username),
    to_user VARCHAR(255),
    content TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    group_id INTEGER REFERENCES rooms(id),
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_messages_from_to ON messages(from_user, to_user);
CREATE INDEX idx_messages_group ON messages(group_id);