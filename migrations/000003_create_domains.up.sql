CREATE TABLE domains (
  id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id        BIGINT UNSIGNED NOT NULL,
  domain         VARCHAR(255) NOT NULL UNIQUE,
  document_root  VARCHAR(512) NOT NULL,
  php_version    VARCHAR(10) NOT NULL DEFAULT '8.3',
  ssl_type       ENUM('none', 'letsencrypt', 'custom') NOT NULL DEFAULT 'none',
  ssl_expires_at TIMESTAMP NULL,
  status         ENUM('active', 'pending', 'error', 'suspended') NOT NULL DEFAULT 'pending',
  created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
