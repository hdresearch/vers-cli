package presenters

import vers "github.com/hdresearch/vers-sdk-go"

type StatusMode int

const (
	StatusList StatusMode = iota
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
	Mode StatusMode
	Head StatusHead
	VM   *vers.Vm
	VMs  []vers.Vm
}
