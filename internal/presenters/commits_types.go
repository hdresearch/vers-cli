package presenters

import vers "github.com/hdresearch/vers-sdk-go"

type CommitsListView struct {
	Commits []vers.CommitInfo
	Total   int64
	Public  bool
}

type CommitParentsView struct {
	CommitID string
	Parents  []vers.CommitListParentsResponse
}

// CommitTagWritten is one entry in CommitCreateView.TagsWritten.
type CommitTagWritten struct {
	Reference string `json:"reference"`
	TagID     string `json:"tag_id"`
}

// CommitCreateView is the JSON/text-mode shape for `vers commit create`.
type CommitCreateView struct {
	CommitID    string             `json:"commit_id"`
	VmID        string             `json:"vm_id"`
	UsedHEAD    bool               `json:"used_head,omitempty"`
	Name        string             `json:"name,omitempty"`
	Description string             `json:"description,omitempty"`
	TagsWritten []CommitTagWritten `json:"tags_written,omitempty"`
	// IsPublic reflects the commit's visibility after any --public publish
	// step. Only populated when --public was supplied; otherwise omitted.
	IsPublic bool `json:"is_public,omitempty"`
}
