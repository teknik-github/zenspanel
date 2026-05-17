CREATE TABLE php_versions (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  version     VARCHAR(10) NOT NULL UNIQUE,
  fpm_socket  VARCHAR(255) NOT NULL,
  enabled     BOOLEAN NOT NULL DEFAULT TRUE,
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO php_versions (version, fpm_socket, enabled) VALUES
  ('8.3', '/run/php/php8.3-fpm.sock', TRUE),
  ('8.2', '/run/php/php8.2-fpm.sock', TRUE),
  ('8.1', '/run/php/php8.1-fpm.sock', TRUE),
  ('7.4', '/run/php/php7.4-fpm.sock', FALSE);
