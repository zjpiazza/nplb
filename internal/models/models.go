package models

import "time"

// DebAsset represents a .deb file from a GitHub release.
type DebAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
	Size        int64  `json:"size"`
}

// Release represents a GitHub release.
type Release struct {
	TagName     string     `json:"tag_name"`
	Name        string     `json:"name"`
	PublishedAt *time.Time `json:"published_at"`
	Assets      []DebAsset `json:"assets"`
}

// DebInfo holds metadata extracted from a .deb file.
type DebInfo struct {
	Package      string `json:"package"`
	Version      string `json:"version"`
	Architecture string `json:"architecture"`
	Depends      string `json:"depends"`
	Description  string `json:"description"`
}

// BuildRequest is the request body for the build endpoint.
type BuildRequest struct {
	Owner string `json:"owner" validate:"required"`
	Repo  string `json:"repo" validate:"required"`
	Limit int    `json:"limit" validate:"min=1,max=100"`
}

// BuildResponse is the response from the build endpoint.
type BuildResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	JobID   string `json:"job_id"`
}
