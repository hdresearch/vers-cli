package presenters

import "time"

type RepoInfo struct {
	RepoID      string    `json:"repo_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
}

type RepoTagInfo struct {
	TagID       string    `json:"tag_id"`
	TagName     string    `json:"tag_name"`
	Reference   string    `json:"reference"`
	CommitID    string    `json:"commit_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RepoListView struct {
	Repositories []RepoInfo
}

type RepoTagListView struct {
	Repository string
	Tags       []RepoTagInfo
}
