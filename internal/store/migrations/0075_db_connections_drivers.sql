-- 0075_db_connections_drivers — widen the Database tool from PostgreSQL
-- only to MySQL, MariaDB and SQLite.
--
-- 0072 created the driver column with an inline (anonymous) CHECK, which
-- PostgreSQL named db_connections_driver_check. Drop and re-add it with the
-- wider engine set. host / username stay NOT NULL: a SQLite connection
-- stores '' for both — its "connection" is a file path in db_name, fenced
-- to the project cwd by the driver at open time — while MySQL/MariaDB use
-- host/port/username like PostgreSQL.
ALTER TABLE db_connections DROP CONSTRAINT IF EXISTS db_connections_driver_check;
ALTER TABLE db_connections ADD CONSTRAINT db_connections_driver_check
    CHECK (driver IN ('postgres', 'mysql', 'mariadb', 'sqlite'));
