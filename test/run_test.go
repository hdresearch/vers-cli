package test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"os/exec"

	"github.com/joho/godotenv"
)


func TestStartCluster(t *testing.T) {
	// Load GO_INSTALL_PATH
	var rootDirPath, err = filepath.Abs("..")
	if err != nil {
		t.Fatal("Failed to resolve root directory. ")
	}
	godotenv.Load(
		fmt.Sprintf(
			"%v/.env",
			rootDirPath, 
		))


	installCli(t)

	login(t)

// 	var got = `Sending request to start cluster...
// Cluster (ID: cluster-abd123) started successfully with root vm 'vm-abd123'.
//  HEAD now points to: vm-dfs34` 

	var got = startCluster(t)
	validateOutput(t, got)
}

func installCli(t *testing.T) {
	var goPath string = os.Getenv("GO_PATH")
	var rootDirPath string
	var err error
	rootDirPath, err = filepath.Abs("..")
	if err != nil {
		t.Fatal("Failed to resolve root directory. ")
	}

	execCommand(
		t, goPath, "install",  fmt.Sprintf("%v/cmd/vers", rootDirPath),
	);
}

func login(t *testing.T) {
	cliPath := getCliPath()

	var apiKey string = os.Getenv(("VERS_API_KEY"))
	execCommand(
		t, cliPath, "login", "--token", apiKey,
	)
}

func startCluster(t *testing.T) string {
	cliPath := getCliPath()
	
	return execCommand(
		t, cliPath, "run", 
	)
}

func getCliPath() string {
	var goInstallPath = strings.TrimRight(os.Getenv("GO_INSTALL_PATH"), "/")
	var cliPath = fmt.Sprintf("%v/vers", goInstallPath)
	return cliPath
}

func validateOutput(t *testing.T, got string) {
	var want = regexp.MustCompile(`Sending request to start cluster...\nCluster \(ID: cluster-\w+\) started successfully with root vm 'vm-\w+'\.\nHEAD now points to: vm-\w+`)

	if !want.MatchString(got) {
		t.Fatalf("Unexpected output. Want = %#q \nGot = %v", want, got)
	}
}

func execCommand(t *testing.T, command string, args ...string) string {

	t.Logf("Executing command %v with args %v…\n", command, args)
	var cmd = exec.Command(
		command, args...,
	);
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe. Got: %v", err)
	}

	var output []byte

	output, err = cmd.Output()

	var stdout = string(output)

	t.Logf("Got output: %v\n", stdout)

	slurp, _ := io.ReadAll(stderr)

	if err != nil {
		t.Fatalf("Failed to execute command. Got error: %v\n", string(slurp))
	}

	return stdout

}

