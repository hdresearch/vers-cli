package presenters

import vers "github.com/hdresearch/vers-sdk-go"

type InfoView struct {
	Metadata *vers.VmMetadataResponse
	UsedHEAD bool
}
