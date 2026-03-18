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
