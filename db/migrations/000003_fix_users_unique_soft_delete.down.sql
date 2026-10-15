DROP INDEX IF EXISTS users_username_active_unique;
DROP INDEX IF EXISTS users_email_active_unique;

ALTER TABLE users ADD CONSTRAINT users_username_unique UNIQUE (username);
ALTER TABLE users ADD CONSTRAINT users_email_unique UNIQUE (email);