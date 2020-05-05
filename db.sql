DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS forums CASCADE;
DROP TABLE IF EXISTS threads CASCADE;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS voteThreads;
DROP TABLE IF EXISTS forumUsers;
CREATE EXTENSION IF NOT EXISTS CITEXT;

DROP TRIGGER IF EXISTS path_updater ON messages;
DROP FUNCTION IF EXISTS updater;

CREATE TABLE users
(
    u_id     BIGSERIAL PRIMARY KEY,
    nickname CITEXT COLLATE "C" NOT NULL UNIQUE,
    fullname VARCHAR(100)       NOT NULL,
    email    CITEXT             NOT NULL UNIQUE,
    about    TEXT
);

CREATE UNIQUE INDEX idx_users_nickname ON users (nickname, email);

CREATE TABLE forums
(
    f_id       BIGSERIAL PRIMARY KEY,
    slug       CITEXT UNIQUE NOT NULL,
    title      TEXT,
    u_nickname CITEXT COLLATE "C" REFERENCES users (nickname) ON DELETE CASCADE
);

CREATE INDEX idx_forums_slug ON forums(slug);


CREATE TABLE threads
(
    t_id       BIGSERIAL PRIMARY KEY,
    slug       CITEXT UNIQUE,
    date       TIMESTAMP WITH TIME ZONE,
    message    TEXT,
    title      TEXT,
    votes      BIGINT DEFAULT 0,
    u_nickname CITEXT COLLATE "C" NOT NULL REFERENCES users (nickname) ON DELETE CASCADE,
    f_slug     CITEXT COLLATE "C" NOT NULL REFERENCES forums (slug) ON DELETE CASCADE
);

CREATE INDEX idx_threads_tidhash ON threads USING hash (t_id);
CREATE INDEX idx_threads_slughash ON threads USING hash (slug);


CREATE TABLE voteThreads
(
    t_id       BIGINT             NOT NULL REFERENCES threads ON DELETE CASCADE,
    counter    INT DEFAULT 0,
    u_nickname CITEXT COLLATE "C" NOT NULL REFERENCES users (nickname) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_voteth_thrnick ON voteThreads (t_id, u_nickname);

CREATE TABLE messages
(
    m_id       BIGSERIAL PRIMARY KEY,
    date       TIMESTAMP WITH TIME ZONE,
    message    TEXT,
    edit       BOOLEAN DEFAULT false,
    parent     BIGINT,
    path       BIGINT[],
    u_nickname CITEXT COLLATE "C" NOT NULL REFERENCES users (nickname) ON DELETE CASCADE,
    f_slug     CITEXT COLLATE "C" NOT NULL REFERENCES forums (slug) ON DELETE CASCADE,
    t_id       BIGINT             NOT NULL REFERENCES threads ON DELETE CASCADE
);

CREATE INDEX idx_messages_mid ON messages (t_id,m_id);

CREATE OR REPLACE FUNCTION updater()
    RETURNS TRIGGER AS
$BODY$
BEGIN
    UPDATE messages SET path = path || NEW.m_id WHERE t_id = NEW.t_id AND m_id = NEW.m_id;
    RETURN NEW;
END;
$BODY$ LANGUAGE plpgsql;

CREATE TRIGGER path_updater
    AFTER INSERT
    ON messages
    FOR EACH ROW
EXECUTE PROCEDURE updater();


CREATE TABLE forumUsers
(
    f_slug     CITEXT COLLATE "C" NOT NULL REFERENCES forums (slug) ON DELETE CASCADE,
    u_nickname CITEXT COLLATE "C" NOT NULL REFERENCES users (nickname) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_forumusers_slugid ON forumUsers (f_slug, u_nickname);

