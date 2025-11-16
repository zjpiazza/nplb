package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zjpiazza/nplb/internal/config"
)

// Queue is an interface for queue operations.
// This abstraction allows easy switching to different queue implementations.
type Queue interface {
	Enqueue(ctx context.Context, taskType string, payload interface{}) error
	Close() error
}

// CloudflareQueue provides a thin client for interacting with Cloudflare Queues via HTTP API.
type CloudflareQueue struct {
	accountID  string
	queueID    string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new Cloudflare Queues client.
// It uses the Cloudflare Queues HTTP API directly for maximum flexibility.
func NewClient(cfg *config.Config) (*CloudflareQueue, error) {
	if cfg.CloudflareAPIToken == "" {
		return nil, fmt.Errorf("CLOUDFLARE_API_TOKEN is required")
	}
	if cfg.R2AccountID == "" {
		return nil, fmt.Errorf("R2_ACCOUNT_ID is required")
	}
	if cfg.QueueName == "" {
		return nil, fmt.Errorf("QUEUE_NAME is required")
	}

	return &CloudflareQueue{
		accountID:  cfg.R2AccountID,
		queueID:    cfg.QueueName,
		apiToken:   cfg.CloudflareAPIToken,
		httpClient: &http.Client{},
	}, nil
}

// Enqueue sends a message to the Cloudflare Queue via HTTP API.
// Reference: https://developers.cloudflare.com/queues/configuration/configure-queues/
func (q *CloudflareQueue) Enqueue(ctx context.Context, taskType string, payload interface{}) error {
	// Construct the message payload
	// The Cloudflare API expects a JSON body with a "body" field containing the message
	message := map[string]interface{}{
		"body": payload,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Construct the API endpoint
	// POST /accounts/<account_id>/queues/<queue_id>/messages
	url := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/accounts/%s/queues/%s/messages",
		q.accountID,
		q.queueID,
	)

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", q.apiToken))
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return fmt.Errorf("queue API returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("queue API error (status %d): %v", resp.StatusCode, errorResponse)
	}

	return nil
}

// Close is a no-op for HTTP-based client.
func (q *CloudflareQueue) Close() error {
	return nil
}
