-- antivirus_enabled: admin can disable antivirus feature per package (V57).
-- DEFAULT TRUE so existing packages keep antivirus enabled after migration.
ALTER TABLE packages
    ADD COLUMN antivirus_enabled BOOLEAN NOT NULL DEFAULT TRUE;
