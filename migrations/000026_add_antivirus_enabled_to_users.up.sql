ALTER TABLE users ADD COLUMN antivirus_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER backup_enabled;
