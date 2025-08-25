package test

import (
	"fmt"
	"os"
	"path/filepath"
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
	startCluster(t)

	// ./test/testdata && \
	// vers run`,

	// var output, err = cmd.Output()

	// if err != nil {
	// 	t.Fatalf("failed to start cluster. Got: %v", err)
	// }

	// var got = string(output)

    // fmt.Printf("Got vers run output: %v", got)

	// var want = regexp.MustCompile(string(golden.Read(t)))
	
	// if !want.MatchString(got) {
	// 	t.Errorf("failed to start cluster. Want = %#q \ngot %v", want, got)
	// }

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

func startCluster(t *testing.T) {
	var goInstallPath = os.Getenv("GO_INSTALL_PATH")
	var cliPath = fmt.Sprintf("%v/vers", goInstallPath)
	
	execCommand(
		t, cliPath, "run", 
	)
}

func execCommand(t *testing.T, command string, args ...string) string {

	t.Logf("Executing command %v with args %v…\n", command, args)
	var cmd = exec.Command(
		command, args...,
	);
	
	var output, err = cmd.Output()

	var stdout = string(output)

	t.Logf("Got output: %v\n", stdout)

	if err != nil {
		t.Fatalf("Failed to execute command. Got error: %v\n", err)
	}

	return stdout

}

