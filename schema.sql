CREATE TABLE IF NOT EXISTS contacts (`name` VARCHAR(256) PRIMARY KEY, `phone` VARCHAR(256), `email` VARCHAR(256));
CREATE TABLE IF NOT EXISTS topics (`channel` VARCHAR(256) PRIMARY KEY, `topic` TEXT);
CREATE TABLE IF NOT EXISTS incidents (`id` INTEGER PRIMARY KEY, `severity` INTEGER, `components` VARCHAR(256), `started_at` DATETIME, `updated_at` DATETIME, status INTEGER, description TEXT);
CREATE TABLE IF NOT EXISTS acls (`command` VARCHAR(256), `identifier` VARCHAR(256), PRIMARY KEY (`command`, `identifier`));