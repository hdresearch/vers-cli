package presenters

type CopyView struct {
	UsedHEAD bool
	HeadID   string
	VMName   string
	Action   string
	Src      string
	Dest     string
}
