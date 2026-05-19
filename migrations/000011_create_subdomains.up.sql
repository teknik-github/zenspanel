-- Subdomains: child of `domains`. Each row = one FQDN nginx vhost
-- under a parent domain owned by the same user. FK to domains
-- cascades on delete so removing the parent torches all subdomains
-- in one statement; the API handler also enumerates them explicitly
-- so it can ask the agent to remove the nginx .conf files first.
CREATE TABLE subdomains (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id           BIGINT UNSIGNED NOT NULL,
  parent_domain_id  BIGINT UNSIGNED NOT NULL,
  subdomain         VARCHAR(63)  NOT NULL,        -- label only, eg "blog"
  fqdn              VARCHAR(253) NOT NULL UNIQUE, -- "blog.example.com"
  document_root     VARCHAR(512) NOT NULL,
  php_version       VARCHAR(10)  NOT NULL DEFAULT '8.3',
  ssl_type          ENUM('none', 'letsencrypt', 'custom') NOT NULL DEFAULT 'none',
  ssl_expires_at    TIMESTAMP NULL,
  status            ENUM('active', 'pending', 'error', 'suspended') NOT NULL DEFAULT 'pending',
  created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_domain_id) REFERENCES domains(id) ON DELETE CASCADE,
  UNIQUE KEY uk_parent_sub (parent_domain_id, subdomain)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
