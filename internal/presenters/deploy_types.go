package presenters

// DeployView contains the data returned from a deploy operation.
type DeployView struct {
	ProjectID string `json:"project_id"`
	VmID      string `json:"vm_id"`
	Status    string `json:"status"`
}
