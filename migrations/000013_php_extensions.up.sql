-- Global catalog: admin controls which extensions are available and
-- whether they are enabled by default. One row per (name, php_version)
-- pair — the same extension may behave differently across PHP versions.
CREATE TABLE php_extensions (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name        VARCHAR(64)  NOT NULL,
  php_version VARCHAR(8)   NOT NULL,
  enabled     BOOLEAN      NOT NULL DEFAULT TRUE,
  created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_ext_ver (name, php_version)
);

-- Per-user override: only rows where the user's choice differs from the
-- global default. Absence = inherit global. ON DELETE CASCADE so
-- deleting a user or removing an extension from the catalog cleans up.
CREATE TABLE user_php_extensions (
  id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id    BIGINT UNSIGNED NOT NULL,
  ext_id     BIGINT UNSIGNED NOT NULL,
  enabled    BOOLEAN         NOT NULL,
  updated_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_user_ext (user_id, ext_id),
  FOREIGN KEY (user_id) REFERENCES users(id)           ON DELETE CASCADE,
  FOREIGN KEY (ext_id)  REFERENCES php_extensions(id)  ON DELETE CASCADE
);

-- Seed common extensions so the admin page is useful out of the box.
-- All enabled=TRUE by default; admin can disable any of them.
INSERT INTO php_extensions (name, php_version, enabled) VALUES
  ('bcmath',    '8.1', TRUE), ('bcmath',    '8.2', TRUE), ('bcmath',    '8.3', TRUE),
  ('curl',      '8.1', TRUE), ('curl',      '8.2', TRUE), ('curl',      '8.3', TRUE),
  ('gd',        '8.1', TRUE), ('gd',        '8.2', TRUE), ('gd',        '8.3', TRUE),
  ('intl',      '8.1', TRUE), ('intl',      '8.2', TRUE), ('intl',      '8.3', TRUE),
  ('mbstring',  '8.1', TRUE), ('mbstring',  '8.2', TRUE), ('mbstring',  '8.3', TRUE),
  ('mysqli',    '8.1', TRUE), ('mysqli',    '8.2', TRUE), ('mysqli',    '8.3', TRUE),
  ('opcache',   '8.1', TRUE), ('opcache',   '8.2', TRUE), ('opcache',   '8.3', TRUE),
  ('pdo_mysql', '8.1', TRUE), ('pdo_mysql', '8.2', TRUE), ('pdo_mysql', '8.3', TRUE),
  ('redis',     '8.1', TRUE), ('redis',     '8.2', TRUE), ('redis',     '8.3', TRUE),
  ('soap',      '8.1', TRUE), ('soap',      '8.2', TRUE), ('soap',      '8.3', TRUE),
  ('xml',       '8.1', TRUE), ('xml',       '8.2', TRUE), ('xml',       '8.3', TRUE),
  ('zip',       '8.1', TRUE), ('zip',       '8.2', TRUE), ('zip',       '8.3', TRUE);
