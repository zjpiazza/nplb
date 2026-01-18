package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hibiken/asynq"
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
func (q *CloudflareQueue) Enqueue(ctx context.Context, taskType string, payload interface{}) error {
	// Construct the message payload with task type
	message := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"body": map[string]interface{}{
					"task_type": taskType,
					"payload":   payload,
				},
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Construct the API endpoint
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

// AsynqQueue provides a client for interacting with Asynq (Redis-based queue).
type AsynqQueue struct {
	client *asynq.Client
}

// NewAsynqClient creates a new Asynq queue client.
func NewAsynqClient(redisAddr, redisPassword string, redisDB int) *AsynqQueue {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	return &AsynqQueue{
		client: client,
	}
}

// Enqueue sends a task to the Asynq queue.
func (q *AsynqQueue) Enqueue(ctx context.Context, taskType string, payload interface{}) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(taskType, jsonPayload)
	_, err = q.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	return nil
}

// Close closes the Asynq client connection.
func (q *AsynqQueue) Close() error {
	return q.client.Close()
}

// NewAsynqServer creates a new Asynq server for processing tasks.
func NewAsynqServer(redisAddr, redisPassword string, redisDB int, concurrency int) *asynq.Server {
	if concurrency <= 0 {
		concurrency = 10
	}

	return asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		},
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				"default":  6,
				"critical": 3,
				"low":      1,
			},
		},
	)
}
