package presenters

// TunnelView holds data for rendering tunnel status.
type TunnelView struct {
	UsedHEAD   bool
	HeadID     string
	VMName     string
	LocalPort  int
	RemoteHost string
	RemotePort int
}
