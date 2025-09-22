package mcp

import "slices"

var registeredTools []string

func trackTool(name string) {
	if !slices.Contains(registeredTools, name) {
		registeredTools = append(registeredTools, name)
	}
}
