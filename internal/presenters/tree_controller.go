package presenters

import (
	"fmt"
	vers "github.com/hdresearch/vers-sdk-go"
)

// RenderTreeController prints any glue text then delegates to existing RenderTree.
func RenderTreeController(cluster vers.APIClusterGetResponseData, headVMID string, findingMsg string) error {
	if findingMsg != "" {
		fmt.Println(findingMsg)
	}
	return RenderTree(cluster, headVMID)
}
