package test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"os/exec"

	"github.com/joho/godotenv"

	"time"
)

func TestVersRun(t *testing.T) {
	loadEnv(t)

	checkEnv(t)

	var installPath string = createTempInstallDir(t)

	var versCliPath string = installVersCli(t, installPath)

	login(t, versCliPath)

	copyVersToml(t, installPath)

	var got = execVersRun(t, versCliPath)

	validateOutput(t, got)

	killNewCluster(t, got, versCliPath)
}

// Load VERS_API_KEY and GO_PATH
func loadEnv(t *testing.T) {
	var rootDirPath, err = filepath.Abs("..")
	if err != nil {
		t.Fatal("Failed to resolve root directory. ")
	}
	godotenv.Load(
		filepath.Join(
			rootDirPath,
			".env",
		),
	)
}

// Validate that the necessary environment variables are set.
func checkEnv(t *testing.T) {
	var goPath, versApiKey string = os.Getenv("GO_PATH"), os.Getenv("VERS_API_KEY")
	var missing []string = []string{}

	if goPath == "" {
		missing = append(missing, "GO_PATH")
	}
	if versApiKey == "" {
		missing = append(missing, "VERS_API_KEY")
	}
	if len(missing) > 0 {
		t.Skipf("Skipping test because environment variables are not set. Missing: %v", missing)
	}
}

// Create a temporary directory for installing the CLI.
// We also use this directory for the vers.toml
func createTempInstallDir(t *testing.T) string {
	var currentDate string = time.Now().Format(time.DateTime)

	tmpDir, err := filepath.Abs("/tmp")
	if err != nil {
		t.Fatal("Failed to resolve tmp directory. ")
	}

	installPath := filepath.Join(tmpDir, "vers-cli-run-test-"+currentDate)

	execCommand(t, "", make(map[string]string), "mkdir", "-p", installPath)
	return installPath
}

// Compile and install the local cmd/vers package
func installVersCli(t *testing.T, installPath string) string {
	var goPath string = filepath.Clean(os.Getenv("GO_PATH"))
	rootDirPath := getRootDirPath(t)

	execCommand(
		t, "", map[string]string{
			"GOBIN": installPath,
		}, goPath, "install", filepath.Join(rootDirPath, "cmd/vers"),
	)
	return filepath.Join(installPath, "vers")
}

// Login to the locally installed CLI
func login(t *testing.T, versCliPath string) {

	var apiKey string = os.Getenv(("VERS_API_KEY"))
	execCommand(
		t, "", make(map[string]string), versCliPath, "login", "--token", apiKey,
	)
}

// Copy the test vers.toml to the testing directory
func copyVersToml(t *testing.T, installPath string) {
	rootDirPath := getRootDirPath(t)
	var versTomlPath string = filepath.Join(rootDirPath, "test", "testdata", "vers.toml")
	var destPath string = filepath.Join(installPath, "vers.toml")

	execCommand(
		t, "", make(map[string]string), "cp", versTomlPath, destPath,
	)
}

// Exec 'vers run'
func execVersRun(t *testing.T, versCliPath string) string {
	return execCommand(
		t, "./testdata", make(map[string]string), versCliPath, "run",
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
func killNewCluster(t *testing.T, got string, versCliPath string) {
	var re = regexp.MustCompile(`Cluster \(ID: (\w+)\) started successfully`)
	var matches = re.FindStringSubmatch(got)

	if len(matches) == 2 {
		var clusterId = matches[1]
		execCommand(t, "", make(map[string]string), versCliPath, "kill", "-c", "-y", clusterId)
	} else {
		t.Errorf("Warning: Failed to extract cluster ID from output: %v. Could not kill cluster", got)
	}
}

// Utility functions

// Get the root directory path
func getRootDirPath(t *testing.T) string {
	var rootDirPath string
	var err error
	rootDirPath, err = filepath.Abs("..")
	if err != nil {
		t.Fatal("Failed to resolve root directory. ")
	}
	return rootDirPath
}

// Executes commands in the specified directory
// Logs stdout and errors
// Command failures are fatal
func execCommand(t *testing.T, dir string, env map[string]string, command string, args ...string) string {

	t.Logf("Executing command %v with args %v…\n", command, args)
	var cmd = exec.Command(
		command, args...,
	)

	cmd.Dir = dir
	for k, v := range env {
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("%s=%s", k, v))
	}

	var output, err = cmd.Output()

	var stdout = string(output)

	t.Logf("Got output: %v\n", stdout)

	if err != nil {
		t.Fatalf("Failed to execute command. Got error: %v\n", err)
	}

	return stdout
}
