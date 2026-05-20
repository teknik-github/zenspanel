-- totp_secret_enc: AES-256-GCM encrypted TOTP secret (V27).
-- totp_recovery_codes: JSON array of bcrypt-hashed single-use codes (V29).
ALTER TABLE users
    ADD COLUMN totp_secret_enc    TEXT         NULL DEFAULT NULL,
    ADD COLUMN totp_enabled       BOOLEAN      NOT NULL DEFAULT FALSE,
    ADD COLUMN totp_recovery_codes TEXT         NULL DEFAULT NULL;
