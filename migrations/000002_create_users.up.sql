CREATE TABLE users (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  username          VARCHAR(64) NOT NULL UNIQUE,
  email             VARCHAR(255) NOT NULL UNIQUE,
  password_hash     VARCHAR(255) NOT NULL,
  role              ENUM('admin', 'user') NOT NULL DEFAULT 'user',
  linux_uid         INT UNSIGNED NOT NULL UNIQUE,
  package_id        BIGINT UNSIGNED NULL,
  status            ENUM('active', 'suspended') NOT NULL DEFAULT 'active',
  terminal_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
  backup_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
  created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (package_id) REFERENCES packages(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
