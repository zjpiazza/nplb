package gpg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// Signer provides GPG signing functionality.
type Signer struct {
	gpgHome  string
	keyEmail string
	logger   *zap.Logger
}

// NewSigner creates a new GPG signer.
func NewSigner(gpgHome, keyEmail string, logger *zap.Logger) *Signer {
	return &Signer{
		gpgHome:  gpgHome,
		keyEmail: keyEmail,
		logger:   logger,
	}
}

// SignDetached creates a detached signature (Release.gpg).
func (s *Signer) SignDetached(content []byte, outputPath string) error {
	// Create a temp file for the content
	tmpFile, err := os.CreateTemp("", "release-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Build gpg command
	args := []string{
		"--batch",
		"--yes",
		"--armor",
		"--detach-sign",
		"--output", outputPath,
	}

	if s.gpgHome != "" {
		args = append([]string{"--homedir", s.gpgHome}, args...)
	}

	if s.keyEmail != "" {
		args = append(args, "--local-user", s.keyEmail)
	}

	args = append(args, tmpFile.Name())

	cmd := exec.Command("gpg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gpg signing failed: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Debug("created detached signature", zap.String("output", outputPath))
	return nil
}

// SignClear creates a clearsigned file (InRelease).
func (s *Signer) SignClear(content []byte, outputPath string) error {
	// Create a temp file for the content
	tmpFile, err := os.CreateTemp("", "release-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Build gpg command
	args := []string{
		"--batch",
		"--yes",
		"--armor",
		"--clearsign",
		"--output", outputPath,
	}

	if s.gpgHome != "" {
		args = append([]string{"--homedir", s.gpgHome}, args...)
	}

	if s.keyEmail != "" {
		args = append(args, "--local-user", s.keyEmail)
	}

	args = append(args, tmpFile.Name())

	cmd := exec.Command("gpg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gpg clearsign failed: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Debug("created clearsigned file", zap.String("output", outputPath))
	return nil
}

// SignDetachedToBytes creates a detached signature and returns it as bytes.
func (s *Signer) SignDetachedToBytes(content []byte) ([]byte, error) {
	tmpOutput, err := os.CreateTemp("", "sig-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	tmpOutput.Close()
	defer os.Remove(tmpOutput.Name())

	if err := s.SignDetached(content, tmpOutput.Name()); err != nil {
		return nil, err
	}

	return os.ReadFile(tmpOutput.Name())
}

// SignClearToBytes creates a clearsigned file and returns it as bytes.
func (s *Signer) SignClearToBytes(content []byte) ([]byte, error) {
	tmpOutput, err := os.CreateTemp("", "inrelease-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	tmpOutput.Close()
	defer os.Remove(tmpOutput.Name())

	if err := s.SignClear(content, tmpOutput.Name()); err != nil {
		return nil, err
	}

	return os.ReadFile(tmpOutput.Name())
}

// ExportPublicKey exports the public key in ASCII armor format.
func (s *Signer) ExportPublicKey() ([]byte, error) {
	args := []string{
		"--batch",
		"--armor",
		"--export",
	}

	if s.gpgHome != "" {
		args = append([]string{"--homedir", s.gpgHome}, args...)
	}

	if s.keyEmail != "" {
		args = append(args, s.keyEmail)
	}

	cmd := exec.Command("gpg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gpg export failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// KeyExists checks if a GPG key exists for the configured email.
func (s *Signer) KeyExists() bool {
	args := []string{
		"--batch",
		"--list-secret-keys",
	}

	if s.gpgHome != "" {
		args = append([]string{"--homedir", s.gpgHome}, args...)
	}

	if s.keyEmail != "" {
		args = append(args, s.keyEmail)
	}

	cmd := exec.Command("gpg", args...)
	return cmd.Run() == nil
}

// GenerateKey generates a new GPG key if one doesn't exist.
func (s *Signer) GenerateKey(name, email string) error {
	if s.KeyExists() {
		s.logger.Info("GPG key already exists", zap.String("email", email))
		return nil
	}

	// Create key generation parameters
	keyParams := fmt.Sprintf(`%%no-protection
Key-Type: RSA
Key-Length: 4096
Subkey-Type: RSA
Subkey-Length: 4096
Name-Real: %s
Name-Email: %s
Expire-Date: 0
%%commit
`, name, email)

	tmpFile, err := os.CreateTemp("", "gpg-params-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(keyParams); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write key params: %w", err)
	}
	tmpFile.Close()

	args := []string{
		"--batch",
		"--gen-key",
	}

	if s.gpgHome != "" {
		// Ensure gpgHome directory exists
		if err := os.MkdirAll(s.gpgHome, 0700); err != nil {
			return fmt.Errorf("failed to create GPG home: %w", err)
		}
		args = append([]string{"--homedir", s.gpgHome}, args...)
	}

	args = append(args, tmpFile.Name())

	cmd := exec.Command("gpg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gpg key generation failed: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Info("generated new GPG key", zap.String("email", email))
	return nil
}

// ImportKey imports a GPG key from a file.
func (s *Signer) ImportKey(keyPath string) error {
	args := []string{
		"--batch",
		"--import",
	}

	if s.gpgHome != "" {
		args = append([]string{"--homedir", s.gpgHome}, args...)
	}

	args = append(args, keyPath)

	cmd := exec.Command("gpg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gpg import failed: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Info("imported GPG key", zap.String("path", keyPath))
	return nil
}

// ImportKeyFromEnv imports a GPG key from an environment variable.
func (s *Signer) ImportKeyFromEnv(envVar string) error {
	keyData := os.Getenv(envVar)
	if keyData == "" {
		return fmt.Errorf("environment variable %s is not set", envVar)
	}

	// Create temp file with key data
	tmpFile, err := os.CreateTemp("", "gpg-key-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(keyData); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write key data: %w", err)
	}
	tmpFile.Close()

	return s.ImportKey(tmpFile.Name())
}

// GetGPGVersion returns the installed GPG version.
func GetGPGVersion() (string, error) {
	cmd := exec.Command("gpg", "--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get GPG version: %w", err)
	}

	lines := strings.Split(stdout.String(), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "", fmt.Errorf("unable to parse GPG version")
}

// EnsureGPGHome ensures the GPG home directory exists with proper permissions.
func EnsureGPGHome(gpgHome string) error {
	if gpgHome == "" {
		return nil
	}

	absPath, err := filepath.Abs(gpgHome)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := os.MkdirAll(absPath, 0700); err != nil {
		return fmt.Errorf("failed to create GPG home directory: %w", err)
	}

	return nil
}
