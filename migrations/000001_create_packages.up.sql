CREATE TABLE packages (
  id                    BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name                  VARCHAR(100) NOT NULL UNIQUE,
  cpu_quota             INT NOT NULL DEFAULT 10000,
  memory_limit          BIGINT NOT NULL DEFAULT 536870912,
  disk_quota            BIGINT NOT NULL DEFAULT 10737418240,
  max_domains           INT NOT NULL DEFAULT 5,
  max_databases         INT NOT NULL DEFAULT 5,
  php_versions_allowed  JSON NOT NULL,
  terminal_enabled      BOOLEAN NOT NULL DEFAULT FALSE,
  backup_enabled        BOOLEAN NOT NULL DEFAULT FALSE,
  created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
