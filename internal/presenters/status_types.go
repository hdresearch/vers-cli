package presenters

import vers "github.com/hdresearch/vers-sdk-go"

type StatusMode int

const (
	StatusList StatusMode = iota
	StatusCluster
	StatusVM
)

type StatusHead struct {
	Show        bool
	Present     bool
	Empty       bool
	ID          string
	DisplayName string
	State       string
}

type StatusView struct {
	Mode     StatusMode
	Head     StatusHead
	Clusters []vers.APIClusterListResponseData
	Cluster  vers.APIClusterGetResponseData
	VM       vers.APIVmGetResponseData
}
