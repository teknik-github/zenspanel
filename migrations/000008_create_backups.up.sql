CREATE TABLE backups (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id     BIGINT UNSIGNED NOT NULL,
  type        ENUM('full', 'db', 'files') NOT NULL,
  status      ENUM('pending', 'running', 'done', 'failed') NOT NULL DEFAULT 'pending',
  file_path   VARCHAR(512) NULL,
  size_bytes  BIGINT NULL,
  error_msg   TEXT NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
