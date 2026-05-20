package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// S3Target holds the connection parameters for an S3-compatible target.
// SecretKeyEnc is AES-256-GCM encrypted (same key as TOTP, from config).
type S3Target struct {
	ID           uint64
	Name         string
	Type         string // "s3", "b2", "gcs", etc.
	Bucket       string
	Prefix       string
	AccessKey    string
	SecretKeyEnc string
	Region       string
	Endpoint     string // empty = AWS default
}

// DecryptSecret decrypts the AES-256-GCM encrypted secret key (V47).
func DecryptSecret(enc string, key []byte) (string, error) {
	ct, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", fmt.Errorf("decode secret: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(ct) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	plain, err := gcm.Open(nil, ct[:gcm.NonceSize()], ct[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plain), nil
}

// UploadS3 uploads filePath to the S3-compatible target using rclone.
// rclone is configured via env vars so no config file is written to disk
// and credentials never appear in process args (V47).
func UploadS3(filePath string, target S3Target, encKey []byte) error {
	secretKey, err := DecryptSecret(target.SecretKeyEnc, encKey)
	if err != nil {
		return fmt.Errorf("decrypt target secret: %w", err)
	}

	// Build the rclone remote name — use a unique per-call name so
	// concurrent uploads don't collide on a shared config file.
	remoteName := fmt.Sprintf("zp-target-%d", target.ID)

	// Destination path: bucket/prefix/filename
	destPath := fmt.Sprintf("%s:%s/%s/%s",
		remoteName,
		target.Bucket,
		target.Prefix,
		filepath.Base(filePath),
	)

	// rclone reads S3 credentials from env vars when the remote type is
	// configured inline. We pass everything via RCLONE_CONFIG_<REMOTE>_*
	// env vars — no file, no args containing secrets (V47).
	env := append(os.Environ(),
		fmt.Sprintf("RCLONE_CONFIG_%s_TYPE=%s", remoteName, rcloneType(target.Type)),
		fmt.Sprintf("RCLONE_CONFIG_%s_ACCESS_KEY_ID=%s", remoteName, target.AccessKey),
		fmt.Sprintf("RCLONE_CONFIG_%s_SECRET_ACCESS_KEY=%s", remoteName, secretKey),
		fmt.Sprintf("RCLONE_CONFIG_%s_REGION=%s", remoteName, target.Region),
	)
	if target.Endpoint != "" {
		env = append(env,
			fmt.Sprintf("RCLONE_CONFIG_%s_ENDPOINT=%s", remoteName, target.Endpoint),
		)
	}

	cmd := exec.Command("rclone", "copyto", filePath, destPath, "--no-traverse")
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rclone upload: %w: %s", err, out)
	}
	return nil
}

// TestConnection verifies that rclone can list the bucket root.
func TestConnection(target S3Target, encKey []byte) error {
	secretKey, err := DecryptSecret(target.SecretKeyEnc, encKey)
	if err != nil {
		return fmt.Errorf("decrypt target secret: %w", err)
	}

	remoteName := fmt.Sprintf("zp-test-%d", target.ID)
	env := append(os.Environ(),
		fmt.Sprintf("RCLONE_CONFIG_%s_TYPE=%s", remoteName, rcloneType(target.Type)),
		fmt.Sprintf("RCLONE_CONFIG_%s_ACCESS_KEY_ID=%s", remoteName, target.AccessKey),
		fmt.Sprintf("RCLONE_CONFIG_%s_SECRET_ACCESS_KEY=%s", remoteName, secretKey),
		fmt.Sprintf("RCLONE_CONFIG_%s_REGION=%s", remoteName, target.Region),
	)
	if target.Endpoint != "" {
		env = append(env,
			fmt.Sprintf("RCLONE_CONFIG_%s_ENDPOINT=%s", remoteName, target.Endpoint),
		)
	}

	cmd := exec.Command("rclone", "lsd", fmt.Sprintf("%s:%s", remoteName, target.Bucket), "--max-depth", "1")
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("connection test failed: %w: %s", err, out)
	}
	return nil
}

func rcloneType(t string) string {
	switch t {
	case "b2":
		return "b2"
	case "gcs":
		return "google cloud storage"
	default:
		return "s3"
	}
}
