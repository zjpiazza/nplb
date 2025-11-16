package github

import (
	"context"
	"strings"

	"github.com/google/go-github/v57/github"
	"github.com/zjpiazza/nplb/internal/models"
	"golang.org/x/oauth2"
)

// Service provides methods for interacting with the GitHub API.
type Service struct {
	client *github.Client
}

// NewService creates a new GitHub service.
func NewService(token string) *Service {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)

	return &Service{
		client: github.NewClient(tc),
	}
}

// GetReleases fetches releases for a given repository.
func (s *Service) GetReleases(ctx context.Context, owner, repo string, limit int) ([]models.Release, error) {
	// TODO: Implement proper pagination and error handling.
	releases, _, err := s.client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{
		PerPage: limit,
	})
	if err != nil {
		return nil, err
	}

	var result []models.Release
	for _, release := range releases {
		var assets []models.DebAsset

		for _, asset := range release.Assets {
			if strings.HasSuffix(*asset.Name, ".deb") {
				assets = append(assets, models.DebAsset{
					Name:        *asset.Name,
					DownloadURL: *asset.BrowserDownloadURL,
					Size:        int64(*asset.Size),
				})
			}
		}

		if len(assets) > 0 {
			result = append(result, models.Release{
				TagName:     *release.TagName,
				Name:        release.GetName(),
				PublishedAt: release.PublishedAt.GetTime(),
				Assets:      assets,
			})
		}
	}

	return result, nil
}
