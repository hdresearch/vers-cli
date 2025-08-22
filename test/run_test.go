package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	cmd "github.com/hdresearch/vers-cli/cmd"
)


func TestStartCluster(t *testing.T) {

	// Load configuration from vers.toml
	config, err := loadTestConfig()

	if err != nil {
		t.Fatalf("failed to load configuration: %q", err)
	}
	
	fmt.Printf("got config: %v", config)

	var  startCluster = func () {
		err = cmd.StartCluster(config, []string{})
		if err != nil {
			t.Errorf("failed to start cluster: %q", err)
		
		}
	}

	got := GetStdOut(startCluster)

    fmt.Printf("Got vers run output: %v", got)

	// var want = regexp.MustCompile(string(golden.Read(t)))
	
	// if !want.MatchString(got) {
	// 	t.Errorf("failed to start cluster. Want = %#q \ngot %v", want, got)
	// }

}

// loadConfig loads the configuration from vers.toml or returns defaults
func loadTestConfig() (*cmd.Config, error) {
	config := cmd.DefaultConfig()

	// Check if vers.toml exists
	if _, err := os.Stat("./testdata/vers.toml"); os.IsNotExist(err) {
		return nil, fmt.Errorf("test vers.toml not found. Exiting.")
	}

	// Read and parse the toml file
	if _, err := toml.DecodeFile("./testdata/vers.toml", config); err != nil {
		return nil, fmt.Errorf("error parsing vers.toml: %w", err)
	}

	return config, nil
}