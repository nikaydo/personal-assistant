-- Bootstrap template for MySQL user and database for localcombineddb.
-- Variables are substituted by Taskfile bootstrap-mysql task:
--   __DB_NAME__, __DB_USER__, __DB_PASSWORD__

CREATE DATABASE IF NOT EXISTS `__DB_NAME__`
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

CREATE USER IF NOT EXISTS '__DB_USER__'@'127.0.0.1' IDENTIFIED BY '__DB_PASSWORD__';
CREATE USER IF NOT EXISTS '__DB_USER__'@'localhost' IDENTIFIED BY '__DB_PASSWORD__';
ALTER USER '__DB_USER__'@'127.0.0.1' IDENTIFIED BY '__DB_PASSWORD__';
ALTER USER '__DB_USER__'@'localhost' IDENTIFIED BY '__DB_PASSWORD__';

GRANT ALL PRIVILEGES ON `__DB_NAME__`.* TO '__DB_USER__'@'127.0.0.1';
GRANT ALL PRIVILEGES ON `__DB_NAME__`.* TO '__DB_USER__'@'localhost';

FLUSH PRIVILEGES;
