package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/zjpiazza/nplb/internal/services/repository"
)

const (
	// TypeBuildRepository is the name of the build repository task.
	TypeBuildRepository = "repository:build"
)

// BuildRepositoryPayload is the payload for the build repository task.
type BuildRepositoryPayload struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Limit       int    `json:"limit"`
	Component   string `json:"component,omitempty"`
	Description string `json:"description,omitempty"`
}

// NewBuildRepositoryTask creates a new build repository task.
func NewBuildRepositoryTask(owner, repo string, limit int) (*asynq.Task, error) {
	payload, err := json.Marshal(BuildRepositoryPayload{Owner: owner, Repo: repo, Limit: limit})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeBuildRepository, payload), nil
}

// TaskHandler handles async tasks.
type TaskHandler struct {
	builder *repository.Builder
	logger  *zap.Logger
}

// NewTaskHandler creates a new task handler.
func NewTaskHandler(builder *repository.Builder, logger *zap.Logger) *TaskHandler {
	return &TaskHandler{
		builder: builder,
		logger:  logger,
	}
}

// HandleBuildRepositoryTask handles the build repository task.
func (h *TaskHandler) HandleBuildRepositoryTask(ctx context.Context, t *asynq.Task) error {
	var p BuildRepositoryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	h.logger.Info("starting repository build task",
		zap.String("owner", p.Owner),
		zap.String("repo", p.Repo),
		zap.Int("limit", p.Limit),
	)

	opts := repository.BuildOptions{
		Owner:       p.Owner,
		Repo:        p.Repo,
		Limit:       p.Limit,
		Component:   p.Component,
		Description: p.Description,
	}

	result, err := h.builder.Build(ctx, opts)
	if err != nil {
		h.logger.Error("repository build failed",
			zap.String("owner", p.Owner),
			zap.String("repo", p.Repo),
			zap.Error(err),
		)
		return fmt.Errorf("build failed: %w", err)
	}

	h.logger.Info("repository build completed",
		zap.String("owner", p.Owner),
		zap.String("repo", p.Repo),
		zap.Int("packages", result.PackagesBuilt),
		zap.Strings("architectures", result.Architectures),
		zap.String("url", result.RepoURL),
	)

	return nil
}

// RegisterHandlers registers all task handlers with the mux.
func (h *TaskHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeBuildRepository, h.HandleBuildRepositoryTask)
}
