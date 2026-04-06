package presenters

import vers "github.com/hdresearch/vers-sdk-go"

type RepoListView struct {
	Repositories []vers.RepositoryInfo
}

type RepoTagListView struct {
	Repository string
	Tags       []vers.RepoTagInfo
}
