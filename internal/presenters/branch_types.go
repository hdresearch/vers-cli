package presenters

type BranchView struct {
	FromID       string
	FromName     string
	NewID        string
	NewIDs       []string
	NewAlias     string
	NewState     string
	CheckoutDone bool
	CheckoutErr  error
	UsedHEAD     bool
}
