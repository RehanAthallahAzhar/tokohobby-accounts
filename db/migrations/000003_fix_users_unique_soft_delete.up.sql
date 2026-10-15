-- Drop old unique constrains
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_unique;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_unique;

-- Create new unique index for active users
CREATE UNIQUE INDEX users_username_active_unique 
ON users (username) 
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX users_email_active_unique 
ON users (email) 
WHERE deleted_at IS NULL;