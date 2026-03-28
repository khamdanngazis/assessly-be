-- Drop users table
DROP TABLE IF EXISTS users CASCADE;

-- Drop UUID extension (only if not used by other tables)
DROP EXTENSION IF EXISTS "uuid-ossp";
