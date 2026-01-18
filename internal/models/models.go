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
	Maintainer   string `json:"maintainer,omitempty"`
	Depends      string `json:"depends,omitempty"`
	Description  string `json:"description,omitempty"`
	Section      string `json:"section,omitempty"`
	Priority     string `json:"priority,omitempty"`
	Homepage     string `json:"homepage,omitempty"`

	// File metadata (populated after parsing)
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MD5sum   string `json:"md5sum"`
	SHA256   string `json:"sha256"`
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

// JobStatus represents the status of a background job.
type JobStatus struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"` // pending, processing, completed, failed
	Owner     string     `json:"owner"`
	Repo      string     `json:"repo"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Error     string     `json:"error,omitempty"`
	Result    *JobResult `json:"result,omitempty"`
}

// JobResult contains the result of a completed build job.
type JobResult struct {
	PackagesBuilt int      `json:"packages_built"`
	RepoURL       string   `json:"repo_url"`
	Architectures []string `json:"architectures"`
}
