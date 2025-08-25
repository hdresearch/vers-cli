package test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"os/exec"

	"github.com/joho/godotenv"
)


func TestVersRun(t *testing.T) {
	loadEnv(t)

	installVersCli(t)

	login(t)

	var got = execVersRun(t)

	validateOutput(t, got)
	
	killNewCluster(t, got)
}

// Load VERS_API_KEY, GO_INSTALL_PATH, and GO_PATH
func loadEnv(t *testing.T) {
	var rootDirPath, err = filepath.Abs("..")
	if err != nil {
		t.Fatal("Failed to resolve root directory. ")
	}
	godotenv.Load(
		fmt.Sprintf(
			"%v/.env",
			rootDirPath,
		),
	)
}

// Compile and install the local cmd/vers package
func installVersCli(t *testing.T) {
	var goPath string = os.Getenv("GO_PATH")
	var rootDirPath string
	var err error
	rootDirPath, err = filepath.Abs("..")
	if err != nil {
		t.Fatal("Failed to resolve root directory. ")
	}

	execCommand(
		t, "", goPath, "install",  fmt.Sprintf("%v/cmd/vers", rootDirPath),
	);
}

// Login to the locally installed CLI
func login(t *testing.T) {
	cliPath := getVersCliPath()

	var apiKey string = os.Getenv(("VERS_API_KEY"))
	execCommand(
		t, "", cliPath, "login", "--token", apiKey,
	)
}

// Exec 'vers run'
func execVersRun(t *testing.T) string {
	cliPath := getVersCliPath()
	
	return execCommand(
		t, "./testdata", cliPath, "run", 
	)
}

// Validate the output of 'vers run'
func validateOutput(t *testing.T, got string) {
	var want = regexp.MustCompile(`Sending request to start cluster...\nCluster \(ID: \w+\) started successfully with root vm '[\w-]+'\.`)

	if !want.MatchString(got) {
		t.Errorf("Unexpected output. Want = %#q \nGot = %v", want, got)
	}
}

// Teardown the newly created cluster
func killNewCluster(t *testing.T, got string) {
	var cliPath = getVersCliPath()
	var re = regexp.MustCompile(`Cluster \(ID: (\w+)\) started successfully`)
	var matches = re.FindStringSubmatch(got)
	
	if len(matches) == 2 {
		var clusterId = matches[1]
		execCommand(t, "", cliPath, "kill", "-c", "-y", clusterId)
	}
}

// Utility functions 

// Get the newly installed CLI executable path
// Expected to be installed to GO_INSTALL_PATH/vers
func getVersCliPath() string {
	var goInstallPath = strings.TrimRight(os.Getenv("GO_INSTALL_PATH"), "/")
	var cliPath = fmt.Sprintf("%v/vers", goInstallPath)
	return cliPath
}

// Executes commands in the specified directory
// Logs stdout and errors
// Command failures are fatal
func execCommand(t *testing.T, dir string, command string,  args ...string) string {

	t.Logf("Executing command %v with args %v…\n", command, args)
	var cmd = exec.Command(
		command, args...,
	);

	cmd.Dir = dir

	var output, err = cmd.Output()

	var stdout = string(output)

	t.Logf("Got output: %v\n", stdout)

	if err != nil {
		t.Fatalf("Failed to execute command. Got error: %v\n", err)
	}

	return stdout
}

