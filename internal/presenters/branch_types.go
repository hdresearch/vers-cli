package presenters

type BranchView struct {
	FromID       string   `json:"from_id"`
	FromName     string   `json:"from_name,omitempty"`
	NewID        string   `json:"new_id,omitempty"`
	NewIDs       []string `json:"new_ids"`
	NewAlias     string   `json:"new_alias,omitempty"`
	NewState     string   `json:"new_state,omitempty"`
	CheckoutDone bool     `json:"checkout_done,omitempty"`
	CheckoutErr  error    `json:"-"`
	UsedHEAD     bool     `json:"used_head,omitempty"`
}
