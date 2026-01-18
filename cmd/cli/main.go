package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zjpiazza/nplb/internal/config"
	"github.com/zjpiazza/nplb/internal/models"
	"github.com/zjpiazza/nplb/internal/services/debian"
	"github.com/zjpiazza/nplb/internal/services/github"
	"github.com/zjpiazza/nplb/internal/services/repository"
	"github.com/zjpiazza/nplb/internal/services/storage"
	"github.com/zjpiazza/nplb/pkg/gpg"
	"go.uber.org/zap"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "build":
		cmdBuild()
	case "trigger":
		cmdTrigger()
	case "gpg-init":
		cmdGPGInit()
	case "version":
		fmt.Printf("NPLB CLI v%s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`NPLB CLI - No Package Left Behind

Usage: nplb-cli <command> [options]

Commands:
  build     Build a repository locally from GitHub releases
  trigger   Trigger a build via the API server
  gpg-init  Initialize GPG key for signing
  version   Print version information
  help      Show this help message

Build Command:
  nplb-cli build <owner> <repo> [limit]
  
  Examples:
    nplb-cli build cli cli 5
    nplb-cli build docker compose

Trigger Command:
  nplb-cli trigger <owner> <repo> [limit]
  
  Requires API server to be running. Set API_URL environment variable.
  Default: http://localhost:8080

GPG Init Command:
  nplb-cli gpg-init <name> <email>
  
  Creates a new GPG key for signing packages.
  Set GPG_HOME environment variable to specify keyring location.

Environment Variables:
  API_URL        API server URL (default: http://localhost:8080)
  GITHUB_TOKEN   GitHub API token (optional, increases rate limits)
  GPG_HOME       GPG keyring directory (default: keys)
  GPG_KEY_EMAIL  Email for GPG signing key
`)
}

func cmdBuild() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: nplb-cli build <owner> <repo> [limit]")
		os.Exit(1)
	}

	owner := os.Args[2]
	repo := os.Args[3]
	limit := 10
	if len(os.Args) > 4 {
		fmt.Sscanf(os.Args[4], "%d", &limit)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Validate required configuration
	if cfg.R2AccountID == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretAccessKey == "" || cfg.R2BucketName == "" {
		fmt.Fprintln(os.Stderr, "Error: R2 storage configuration is required")
		fmt.Fprintln(os.Stderr, "Set R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, and R2_BUCKET_NAME")
		os.Exit(1)
	}

	fmt.Printf("Building repository for %s/%s (limit: %d releases)\n", owner, repo, limit)

	// Initialize services
	githubSvc := github.NewService(cfg.GitHubToken, logger)
	debianSvc := debian.NewService(logger)

	storageSvc, err := storage.NewR2Storage(
		cfg.R2AccountID,
		cfg.R2AccessKeyID,
		cfg.R2SecretAccessKey,
		cfg.R2BucketName,
		cfg.R2PublicURL,
		logger,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}

	gpgSigner := gpg.NewSigner(cfg.GPGHome, cfg.GPGKeyEmail, logger)

	// Create output directory
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize builder
	builder := repository.NewBuilder(
		githubSvc,
		debianSvc,
		storageSvc,
		gpgSigner,
		logger,
		cfg.OutputDir,
		cfg.DefaultCodename,
	)

	// Build repository
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	result, err := builder.Build(ctx, repository.BuildOptions{
		Owner: owner,
		Repo:  repo,
		Limit: limit,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nBuild completed successfully!")
	fmt.Printf("  Packages built: %d\n", result.PackagesBuilt)
	fmt.Printf("  Architectures: %s\n", strings.Join(result.Architectures, ", "))
	fmt.Printf("  Repository URL: %s\n", result.RepoURL)
	fmt.Println("\nTo use this repository, add the following to your sources.list:")
	fmt.Printf("  deb %s %s main\n", strings.TrimSuffix(result.RepoURL, "/dists/"+cfg.DefaultCodename), cfg.DefaultCodename)
}

func cmdTrigger() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: nplb-cli trigger <owner> <repo> [limit]")
		os.Exit(1)
	}

	owner := os.Args[2]
	repo := os.Args[3]
	limit := 10
	if len(os.Args) > 4 {
		fmt.Sscanf(os.Args[4], "%d", &limit)
	}

	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	fmt.Printf("Triggering build for %s/%s via API (%s)\n", owner, repo, apiURL)

	// Create request
	reqBody := models.BuildRequest{
		Owner: owner,
		Repo:  repo,
		Limit: limit,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal request: %v\n", err)
		os.Exit(1)
	}

	// Send request
	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/build", apiURL),
		"application/json",
		strings.NewReader(string(jsonBody)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Parse response
	var result models.BuildResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("Build triggered successfully!\n")
		fmt.Printf("  Status: %s\n", result.Status)
		fmt.Printf("  Message: %s\n", result.Message)
		if result.JobID != "" && result.JobID != "not-implemented" {
			fmt.Printf("  Job ID: %s\n", result.JobID)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Build trigger failed (HTTP %d)\n", resp.StatusCode)
		os.Exit(1)
	}
}

func cmdGPGInit() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: nplb-cli gpg-init <name> <email>")
		os.Exit(1)
	}

	name := os.Args[2]
	email := os.Args[3]

	gpgHome := os.Getenv("GPG_HOME")
	if gpgHome == "" {
		gpgHome = "keys"
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Ensure GPG home exists
	if err := gpg.EnsureGPGHome(gpgHome); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create GPG home: %v\n", err)
		os.Exit(1)
	}

	signer := gpg.NewSigner(gpgHome, email, logger)

	// Check if key already exists
	if signer.KeyExists() {
		fmt.Printf("GPG key for %s already exists in %s\n", email, gpgHome)
		return
	}

	fmt.Printf("Generating GPG key for %s <%s>...\n", name, email)

	if err := signer.GenerateKey(name, email); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("GPG key generated successfully!")

	// Export public key
	pubKey, err := signer.ExportPublicKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to export public key: %v\n", err)
		return
	}

	pubKeyPath := fmt.Sprintf("%s/public.gpg", gpgHome)
	if err := os.WriteFile(pubKeyPath, pubKey, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save public key: %v\n", err)
		return
	}

	fmt.Printf("Public key saved to %s\n", pubKeyPath)
	fmt.Println("\nTo import this key on client machines:")
	fmt.Printf("  curl -fsSL <your-repo-url>/public.gpg | sudo gpg --dearmor -o /etc/apt/keyrings/<name>.gpg\n")
}
