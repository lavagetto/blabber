CREATE TABLE contacts (`name` VARCHAR(256) PRIMARY KEY, `phone` VARCHAR(256), `email` VARCHAR(256));
CREATE TABLE topics (`channel` VARCHAR(256) PRIMARY KEY, `topic` TEXT);
CREATE TABLE incidents (`id` INTEGER PRIMARY KEY, `severity` INTEGER, `components` VARCHAR(256), `started_at` DATETIME, `updated_at` DATETIME, status INTEGER, description TEXT, `document_id` VARCHAR(256));
CREATE TABLE acls (`command` VARCHAR(256), `identifier` VARCHAR(256), PRIMARY KEY (`command`, `identifier`));
