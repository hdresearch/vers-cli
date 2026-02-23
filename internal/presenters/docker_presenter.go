package presenters

import (
	"fmt"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
)

// DockerRunView contains the result of a docker run command
type DockerRunView struct {
	VMID          string
	VMAlias       string
	BaseImage     string
	ExposedPorts  []string
	StartCommand  []string
	SetupComplete bool
	Running       bool
}

// RenderDockerRun renders the docker run result
func RenderDockerRun(a *app.App, v DockerRunView) {
	fmt.Println()
	fmt.Println("✅ Dockerfile executed successfully on Vers VM")
	fmt.Println()

	// VM Info
	fmt.Printf("   VM ID:     %s\n", v.VMID)
	if v.VMAlias != "" {
		fmt.Printf("   Alias:     %s\n", v.VMAlias)
	}
	fmt.Printf("   Base:      %s (translated to Vers VM)\n", v.BaseImage)

	// Ports
	if len(v.ExposedPorts) > 0 {
		fmt.Printf("   Ports:     %s\n", strings.Join(v.ExposedPorts, ", "))
		fmt.Println()
		fmt.Println("   🌐 To access exposed ports, use:")
		fmt.Printf("      vers tunnel %s <port>\n", v.VMID)
	}

	// Status
	fmt.Println()
	if v.Running {
		fmt.Println("   Status: Application running in background")
		fmt.Println("   View logs: vers execute " + v.VMID + " cat /tmp/app.log")
	} else if v.SetupComplete {
		fmt.Println("   Status: Setup complete")
	}

	// Next steps
	fmt.Println()
	fmt.Println("   Next steps:")
	fmt.Printf("      vers connect %s     # SSH into the VM\n", v.VMID)
	fmt.Printf("      vers kill %s        # Stop the VM\n", v.VMID)
	if len(v.ExposedPorts) > 0 {
		fmt.Printf("      vers tunnel %s %s  # Forward port\n", v.VMID, v.ExposedPorts[0])
	}
}

// DockerBuildView contains the result of a docker build command
type DockerBuildView struct {
	VMID       string
	CommitID   string
	BaseImage  string
	NumLayers  int
	TotalSteps int
}

// RenderDockerBuild renders the docker build result
func RenderDockerBuild(a *app.App, v DockerBuildView) {
	fmt.Println()
	fmt.Println("✅ Docker image built as Vers snapshot")
	fmt.Println()
	fmt.Printf("   Commit ID: %s\n", v.CommitID)
	fmt.Printf("   Base:      %s\n", v.BaseImage)
	fmt.Printf("   Layers:    %d steps executed\n", v.NumLayers)
	fmt.Println()
	fmt.Println("   To run this image:")
	fmt.Printf("      vers checkout %s\n", v.CommitID)
}
