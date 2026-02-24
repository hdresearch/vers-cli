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
	Tag        string
}

// RenderDockerBuild renders the docker build result
func RenderDockerBuild(a *app.App, v DockerBuildView) {
	fmt.Println()
	fmt.Println("✅ Docker image built as Vers snapshot")
	fmt.Println()
	fmt.Printf("   Commit ID: %s\n", v.CommitID)
	if v.Tag != "" {
		fmt.Printf("   Tag:       %s\n", v.Tag)
	}
	fmt.Printf("   Base:      %s\n", v.BaseImage)
	fmt.Printf("   Layers:    %d steps executed\n", v.TotalSteps)
	fmt.Println()
	fmt.Println("   To run this image:")
	fmt.Printf("      vers checkout %s\n", v.CommitID)
	if v.Tag != "" {
		fmt.Println()
		fmt.Println("   Or by tag:")
		fmt.Printf("      vers checkout image:%s\n", v.Tag)
	}
}

// ComposeServiceView contains the result of starting a single compose service
type ComposeServiceView struct {
	Name          string
	VMID          string
	VMAlias       string
	Ports         []string
	Running       bool
	SetupComplete bool
	Error         string
}

// DockerComposeView contains the result of a docker compose up command
type DockerComposeView struct {
	ProjectName   string
	TotalServices int
	Services      []ComposeServiceView
}

// RenderDockerComposeUp renders the docker compose up result
func RenderDockerComposeUp(a *app.App, v DockerComposeView) {
	fmt.Println()
	fmt.Printf("✅ Docker Compose project '%s' started on Vers VMs\n", v.ProjectName)
	fmt.Println()

	// Count successful services
	successCount := 0
	for _, svc := range v.Services {
		if svc.Error == "" && svc.VMID != "" {
			successCount++
		}
	}

	fmt.Printf("   Services: %d/%d running\n", successCount, v.TotalServices)
	fmt.Println()

	// List services
	fmt.Println("   ┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("   │ Service           │ VM ID              │ Status             │")
	fmt.Println("   ├───────────────────┼────────────────────┼────────────────────┤")

	for _, svc := range v.Services {
		status := "❌ Failed"
		if svc.Error == "" {
			if svc.Running {
				status = "▶️  Running"
			} else if svc.SetupComplete {
				status = "✅ Ready"
			}
		}

		vmID := svc.VMID
		if len(vmID) > 18 {
			vmID = vmID[:15] + "..."
		}
		name := svc.Name
		if len(name) > 17 {
			name = name[:14] + "..."
		}

		fmt.Printf("   │ %-17s │ %-18s │ %-18s │\n", name, vmID, status)
	}

	fmt.Println("   └─────────────────────────────────────────────────────────────┘")

	// Show errors if any
	hasErrors := false
	for _, svc := range v.Services {
		if svc.Error != "" {
			if !hasErrors {
				fmt.Println()
				fmt.Println("   Errors:")
				hasErrors = true
			}
			fmt.Printf("   - %s: %s\n", svc.Name, svc.Error)
		}
	}

	// Show exposed ports
	hasPorts := false
	for _, svc := range v.Services {
		if len(svc.Ports) > 0 && svc.Error == "" {
			if !hasPorts {
				fmt.Println()
				fmt.Println("   🌐 To access exposed ports:")
				hasPorts = true
			}
			for _, port := range svc.Ports {
				fmt.Printf("      vers tunnel %s %s   # %s\n", svc.VMAlias, port, svc.Name)
			}
		}
	}

	// Next steps
	fmt.Println()
	fmt.Println("   Next steps:")
	if len(v.Services) > 0 && v.Services[0].VMAlias != "" {
		fmt.Printf("      vers connect %s     # SSH into a service\n", v.Services[0].VMAlias)
	}
	fmt.Printf("      vers status                 # List all VMs\n")
}
