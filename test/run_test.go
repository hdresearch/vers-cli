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


func TestStartCluster(t *testing.T) {
	var rootDirPath, err = filepath.Abs("..")
	if err != nil {
		t.Fatal("Failed to resolve root directory. ")
	}
	// Load VERS_API_KEY, GO_INSTALL_PATH, and GO_PATH
	godotenv.Load(
		fmt.Sprintf(
			"%v/.env",
			rootDirPath, 
		),
	)


	installCli(t)

	login(t)

// 	var got = `Sending request to start cluster...
// Cluster (ID: 5R7XSWVN8XRTD5Z9cCyp2S) started successfully with root vm 'b6bd5680-d942-4b64-b805-8c43ae5955f0'.
// Warning: .vers directory not found. Run 'vers init' first.` 

	var got = startCluster(t)
	validateOutput(t, got)
	killCluster(t, got)
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
		t, "", goPath, "install",  fmt.Sprintf("%v/cmd/vers", rootDirPath),
	);
}

func login(t *testing.T) {
	cliPath := getCliPath()

	var apiKey string = os.Getenv(("VERS_API_KEY"))
	execCommand(
		t, "", cliPath, "login", "--token", apiKey,
	)
}

func startCluster(t *testing.T) string {
	cliPath := getCliPath()
	
	return execCommand(
		t, "./testdata", cliPath, "run", 
	)
}


func validateOutput(t *testing.T, got string) {
	var want = regexp.MustCompile(`Sending request to start cluster...\nCluster \(ID: \w+\) started successfully with root vm '[\w-]+'\.`)

	if !want.MatchString(got) {
		t.Errorf("Unexpected output. Want = %#q \nGot = %v", want, got)
	}
}

func killCluster(t *testing.T, got string) {
	var cliPath = getCliPath()
	var re = regexp.MustCompile(`Cluster \(ID: (\w+)\) started successfully`)
	var matches = re.FindStringSubmatch(got)
	
	if len(matches) == 2 {
		var clusterId = matches[1]
		execCommand(t, "", cliPath, "kill", "-c", "-y", clusterId)
	}
}

func getCliPath() string {
	var goInstallPath = strings.TrimRight(os.Getenv("GO_INSTALL_PATH"), "/")
	var cliPath = fmt.Sprintf("%v/vers", goInstallPath)
	return cliPath
}

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

