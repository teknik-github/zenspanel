-- Per-user shell PHP version. Used by the terminal handler when it
-- spawns a shell — the agent symlinks ~/bin/php → /usr/bin/php<ver>
-- and writes a ~/bin/composer wrapper that invokes that same version.
-- Default 8.3 matches the installer's auto-started FPM unit, so
-- existing users get a sane default without manual migration.
ALTER TABLE users ADD COLUMN php_version VARCHAR(8) NOT NULL DEFAULT '8.3';
