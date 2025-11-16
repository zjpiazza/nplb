package tasks

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	// "github.com/zjpiazza/nplb/internal/config"
	// "github.com/zjpiazza/nplb/internal/services/github"
	// "github.com/zjpiazza/nplb/internal/services/storage"
)

const (
	// TypeBuildRepository is the name of the build repository task.
	TypeBuildRepository = "repository:build"
)

// BuildRepositoryPayload is the payload for the build repository task.
type BuildRepositoryPayload struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Limit int    `json:"limit"`
}

// NewBuildRepositoryTask creates a new build repository task.
func NewBuildRepositoryTask(owner, repo string, limit int) (*asynq.Task, error) {
	payload, err := json.Marshal(BuildRepositoryPayload{Owner: owner, Repo: repo, Limit: limit})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeBuildRepository, payload), nil
}

// HandleBuildRepositoryTask handles the build repository task.
func HandleBuildRepositoryTask(ctx context.Context, t *asynq.Task /*, githubService *github.Service, storageService *storage.Service, cfg *config.Config, logger *zap.Logger*/) error {
	var p BuildRepositoryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// logger.Info("Starting repository build", zap.String("owner", p.Owner), zap.String("repo", p.Repo))

	// 1. Fetch releases from GitHub
	// 2. Download .deb files
	// 3. Parse .deb files
	// 4. Generate repository metadata
	// 5. Upload repository to R2

	// logger.Info("Finished repository build", zap.String("owner", p.Owner), zap.String("repo", p.Repo))

	return nil
}
