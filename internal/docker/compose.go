package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents a docker-compose.yml file
type ComposeFile struct {
	Version  string                    `yaml:"version"`
	Services map[string]ComposeService `yaml:"services"`
}

// ComposeService represents a service in docker-compose.yml
type ComposeService struct {
	Image       string            `yaml:"image"`
	Build       *ComposeBuild     `yaml:"build"`
	Ports       []string          `yaml:"ports"`
	Environment interface{}       `yaml:"environment"` // Can be []string or map[string]string
	DependsOn   []string          `yaml:"depends_on"`
	Command     interface{}       `yaml:"command"` // Can be string or []string
	Volumes     []string          `yaml:"volumes"`
	WorkingDir  string            `yaml:"working_dir"`
	EnvFile     interface{}       `yaml:"env_file"` // Can be string or []string
	Labels      map[string]string `yaml:"labels"`
	MemLimit    string            `yaml:"mem_limit"`
	CPUs        float64           `yaml:"cpus"`
}

// ComposeBuild represents the build configuration for a service
type ComposeBuild struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

// UnmarshalYAML handles the case where build can be a string or struct
func (b *ComposeBuild) UnmarshalYAML(value *yaml.Node) error {
	// Try string first (short form: build: ./path)
	var strVal string
	if err := value.Decode(&strVal); err == nil {
		b.Context = strVal
		b.Dockerfile = "Dockerfile"
		return nil
	}

	// Otherwise try struct form
	type buildAlias ComposeBuild
	var build buildAlias
	if err := value.Decode(&build); err != nil {
		return err
	}
	*b = ComposeBuild(build)

	// Set defaults
	if b.Dockerfile == "" {
		b.Dockerfile = "Dockerfile"
	}

	return nil
}

// ParsedService contains a fully resolved service configuration
type ParsedService struct {
	Name         string
	Dockerfile   string            // Path to Dockerfile (if build context)
	BuildContext string            // Build context directory
	Image        string            // Base image (if no build)
	Ports        []string          // Port mappings
	Environment  map[string]string // Environment variables
	DependsOn    []string          // Services this depends on
	Command      []string          // Command to run
	WorkingDir   string            // Working directory
	MemSizeMib   int64             // Memory in MiB
	VcpuCount    int64             // Number of vCPUs
}

// ParseComposeFile reads and parses a docker-compose.yml file
func ParseComposeFile(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	if len(compose.Services) == 0 {
		return nil, fmt.Errorf("compose file has no services defined")
	}

	return &compose, nil
}

// ParseServices resolves all services in the compose file to ParsedService structs
func (c *ComposeFile) ParseServices(baseDir string) ([]ParsedService, error) {
	var services []ParsedService

	for name, svc := range c.Services {
		parsed := ParsedService{
			Name:       name,
			Image:      svc.Image,
			Ports:      svc.Ports,
			DependsOn:  svc.DependsOn,
			WorkingDir: svc.WorkingDir,
			MemSizeMib: 1024, // Default
			VcpuCount:  2,    // Default
		}

		// Parse build configuration
		if svc.Build != nil {
			buildContext := svc.Build.Context
			if !filepath.IsAbs(buildContext) {
				buildContext = filepath.Join(baseDir, buildContext)
			}
			parsed.BuildContext = buildContext
			parsed.Dockerfile = filepath.Join(buildContext, svc.Build.Dockerfile)
		}

		// Parse environment variables
		parsed.Environment = parseEnvironment(svc.Environment)

		// Parse command
		parsed.Command = parseCommand(svc.Command)

		// Parse memory limit (e.g., "512m", "1g")
		if svc.MemLimit != "" {
			parsed.MemSizeMib = parseMemoryLimit(svc.MemLimit)
		}

		// Parse CPU limit
		if svc.CPUs > 0 {
			parsed.VcpuCount = int64(svc.CPUs)
			if parsed.VcpuCount < 1 {
				parsed.VcpuCount = 1
			}
		}

		services = append(services, parsed)
	}

	// Sort by dependency order (topological sort)
	sorted, err := topologicalSort(services)
	if err != nil {
		return nil, err
	}

	return sorted, nil
}

// parseEnvironment handles the various formats for environment variables
func parseEnvironment(env interface{}) map[string]string {
	result := make(map[string]string)

	switch e := env.(type) {
	case []interface{}:
		// Format: ["KEY=value", "KEY2=value2"]
		for _, item := range e {
			if s, ok := item.(string); ok {
				if idx := indexOf(s, '='); idx > 0 {
					result[s[:idx]] = s[idx+1:]
				} else {
					// Just key, value from host env
					result[s] = os.Getenv(s)
				}
			}
		}
	case map[string]interface{}:
		// Format: {KEY: value, KEY2: value2}
		for k, v := range e {
			if v == nil {
				result[k] = os.Getenv(k)
			} else {
				result[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return result
}

// parseCommand handles the various formats for command
func parseCommand(cmd interface{}) []string {
	switch c := cmd.(type) {
	case string:
		return []string{c}
	case []interface{}:
		var result []string
		for _, item := range c {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// parseMemoryLimit parses memory limits like "512m", "1g", "1024"
func parseMemoryLimit(limit string) int64 {
	if len(limit) == 0 {
		return 1024
	}

	var multiplier int64 = 1
	suffix := limit[len(limit)-1]
	numStr := limit

	switch suffix {
	case 'g', 'G':
		multiplier = 1024
		numStr = limit[:len(limit)-1]
	case 'm', 'M':
		multiplier = 1
		numStr = limit[:len(limit)-1]
	case 'k', 'K':
		multiplier = 1 // Round up to 1 MiB minimum
		numStr = limit[:len(limit)-1]
	}

	var num int64
	fmt.Sscanf(numStr, "%d", &num)
	result := num * multiplier
	if result < 256 {
		return 256 // Minimum
	}
	return result
}

// indexOf finds the index of a rune in a string
func indexOf(s string, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

// topologicalSort sorts services by dependency order
func topologicalSort(services []ParsedService) ([]ParsedService, error) {
	// Build adjacency list
	serviceMap := make(map[string]*ParsedService)
	for i := range services {
		serviceMap[services[i].Name] = &services[i]
	}

	// Kahn's algorithm
	inDegree := make(map[string]int)
	for _, svc := range services {
		if _, exists := inDegree[svc.Name]; !exists {
			inDegree[svc.Name] = 0
		}
		for _, dep := range svc.DependsOn {
			inDegree[svc.Name]++
			if _, exists := inDegree[dep]; !exists {
				inDegree[dep] = 0
			}
		}
	}

	// Find all nodes with no incoming edges
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // Stable sort for determinism

	var result []ParsedService
	for len(queue) > 0 {
		// Pop first element
		name := queue[0]
		queue = queue[1:]

		if svc, exists := serviceMap[name]; exists {
			result = append(result, *svc)
		}

		// Decrease in-degree for dependents
		for _, svc := range services {
			for _, dep := range svc.DependsOn {
				if dep == name {
					inDegree[svc.Name]--
					if inDegree[svc.Name] == 0 {
						queue = append(queue, svc.Name)
						sort.Strings(queue)
					}
				}
			}
		}
	}

	// Check for cycles
	if len(result) != len(services) {
		return nil, fmt.Errorf("circular dependency detected in compose services")
	}

	return result, nil
}

// GetServiceNames returns all service names
func (c *ComposeFile) GetServiceNames() []string {
	var names []string
	for name := range c.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
