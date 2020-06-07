ALTER SYSTEM SET checkpoint_completion_target = '0.9';
ALTER SYSTEM SET wal_buffers = '6912kB';
ALTER SYSTEM SET default_statistics_target = '100';
ALTER SYSTEM SET random_page_cost = '1.1';
ALTER SYSTEM SET effective_io_concurrency = '200';

DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS forums CASCADE;
DROP TABLE IF EXISTS threads CASCADE;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS voteThreads;
DROP TABLE IF EXISTS forumUsers;
CREATE EXTENSION IF NOT EXISTS CITEXT;

DROP TRIGGER IF EXISTS path_updater ON messages;
DROP FUNCTION IF EXISTS updater;

DROP TRIGGER IF EXISTS fu_updater ON messages;
DROP FUNCTION IF EXISTS fupdater;

CREATE UNLOGGED TABLE users
(
    u_id     BIGSERIAL PRIMARY KEY,
    nickname CITEXT COLLATE "C" UNIQUE,
    fullname VARCHAR(100) NOT NULL,
    email    CITEXT       NOT NULL UNIQUE,
    about    TEXT
);

CREATE INDEX idx_users_nickname ON users (email);
CREATE INDEX idx_users_all ON users (nickname, fullname, email, about);
CLUSTER users USING idx_users_all;

CREATE UNLOGGED TABLE forums
(
    f_id            BIGSERIAL PRIMARY KEY,
    slug            CITEXT UNIQUE NOT NULL,
    title           TEXT,
    message_counter BIGINT DEFAULT 0,
    thread_counter  BIGINT DEFAULT 0,
    u_nickname      CITEXT COLLATE "C" REFERENCES users (nickname) ON DELETE CASCADE
);

CREATE INDEX idx_forums_slug ON forums (slug);
CREATE INDEX idx_forums_slug_hash ON forums USING hash (slug);
CLUSTER forums USING idx_forums_slug_hash;
CREATE INDEX idx_forums_all ON forums (slug, title, u_nickname, message_counter, thread_counter);


CREATE UNLOGGED TABLE threads
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


CREATE INDEX idx_threads_fslugdate ON threads (f_slug, date);
CLUSTER threads USING idx_threads_fslugdate;
CREATE INDEX idx_threads_slug ON threads (slug);
CREATE INDEX idx_threads_slughash ON threads USING hash (slug);
CREATE INDEX idx_threads_tidhash ON threads USING hash (t_id);
CREATE INDEX idx_threads_all ON threads (t_id, date, message, title, votes, slug, f_slug, u_nickname);


CREATE UNLOGGED TABLE voteThreads
(
    vt_id      BIGSERIAL,
    t_id       BIGINT             NOT NULL REFERENCES threads ON DELETE CASCADE,
    counter    INT DEFAULT 0,
    u_nickname CITEXT COLLATE "C" NOT NULL REFERENCES users (nickname) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_voteth_thrnick ON voteThreads USING btree (t_id, u_nickname);

CREATE UNLOGGED TABLE messages
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

CREATE INDEX idx_messages_tid_mid ON messages (t_id, m_id);
CREATE INDEX idx_messages_parent_tree_tid_parent ON messages (t_id, m_id) WHERE parent = 0;
CREATE INDEX idx_messages_path_1 ON messages (t_id, (path[1]), path);
CLUSTER messages USING idx_messages_path_1;
CREATE INDEX idx_messages_tid_path ON messages (t_id, path);
CREATE INDEX idx_messages_path ON messages (path, m_id);
CREATE INDEX idx_messages_all ON messages (m_id, date, message, edit, parent, u_nickname, t_id, f_slug);

CREATE UNLOGGED TABLE forumUsers
(
    f_slug     CITEXT COLLATE "C" NOT NULL REFERENCES forums (slug) ON DELETE CASCADE,
    u_nickname CITEXT COLLATE "C" NOT NULL REFERENCES users (nickname) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_forumusers_slug_nick ON forumUsers (f_slug, u_nickname);
CLUSTER forumUsers USING idx_forumusers_slug_nick;
CREATE INDEX idx_forumusers_nick ON forumUsers (u_nickname);

CREATE OR REPLACE FUNCTION updater()
    RETURNS TRIGGER AS
$BODY$
DECLARE
    parent_path         BIGINT[];
    first_parent_thread INT;
BEGIN
    IF (NEW.parent = 0) THEN
        NEW.path := array_append(NEW.path, NEW.m_id);
    ELSE
        SELECT t_id, path
        FROM messages
        WHERE t_id = NEW.t_id AND m_id = NEW.parent
        INTO first_parent_thread , parent_path;
        IF NOT FOUND OR first_parent_thread != NEW.t_id THEN
            RAISE EXCEPTION 'Parent post was created in another thread' USING ERRCODE = '00404';
        END IF;

        NEW.path := parent_path || NEW.m_id;
    END IF;
    RETURN NEW;
END;
$BODY$ LANGUAGE plpgsql;

CREATE TRIGGER u_updater
    BEFORE INSERT
    ON messages
    FOR EACH ROW
EXECUTE PROCEDURE updater();


