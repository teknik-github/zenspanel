CREATE TABLE antivirus_alerts (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id      BIGINT UNSIGNED NOT NULL,
  path         VARCHAR(1024)   NOT NULL,
  threat       VARCHAR(256)    NOT NULL,
  detected_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_user_detected (user_id, detected_at)
);
