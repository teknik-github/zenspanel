ALTER TABLE users
    DROP COLUMN totp_secret_enc,
    DROP COLUMN totp_enabled,
    DROP COLUMN totp_recovery_codes;
