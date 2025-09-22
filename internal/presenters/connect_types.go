package presenters

type ConnectView struct {
	UsedHEAD   bool
	HeadID     string
	VMName     string
	SSHHost    string
	SSHPort    string
	LocalRoute bool
}
