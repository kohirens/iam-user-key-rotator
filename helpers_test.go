package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"
)

// Used for running the application's main function from other test.
func runAppMain() {
	args := strings.Split(os.Getenv(subCmdFlags), " ")
	os.Args = append([]string{os.Args[0]}, args...)

	// Debug stmt
	fmt.Printf("\nos args = %v\n", os.Args)

	main()
}

// getTestBinCmd return a command to run your app (test) binary directly; `TestMain`, will be run automatically.
func getTestBinCmd(args []string) *exec.Cmd {
	var cmd *exec.Cmd
	// call the generated test binary directly
	// Have it the function runAppMain.
	if len(args) > 0 {
		cmd = exec.Command(os.Args[0], "-args", strings.Join(args, " "))
	} else {
		cmd = exec.Command(os.Args[0])
	}
	// Run in the context of the source directory.
	_, filename, _, _ := runtime.Caller(0)
	cmd.Dir = path.Dir(filename)
	// Set an environment variable
	// 1. Only exist for the life of the test that calls this function.
	// 2. Passes arguments/flag to your app
	// 3. Lets TestMain know when to run the main function.
	subEnvVar := subCmdFlags + "=" + strings.Join(args, " ")
	cmd.Env = append(os.Environ(), subEnvVar)

	return cmd
}

// quiet Prints output to the OS null space.
func quiet() func() {
	null, _ := os.Open(os.DevNull)
	sOut := os.Stdout
	sErr := os.Stderr
	os.Stdout = null
	os.Stderr = null
	log.SetOutput(null)
	return func() {
		defer null.Close()
		os.Stdout = sOut
		os.Stderr = sErr
		log.SetOutput(os.Stderr)
	}
}

// showCmdOutput display output captured during a run of a sub-command. Just add `-v` as in `go test -v`
func showCmdOutput(cmdOut []byte, cmdErr error) {
	if !testing.Verbose() {
		return
	}

	if cmdOut != nil {
		fmt.Printf("\nBEGIN sub-command out:\n%v", string(cmdOut))
		fmt.Print("END sub-command\n")
	}

	if cmdErr != nil {
		fmt.Printf("\nBEGIN sub-command stderr: %v\n", cmdErr.Error())
		fmt.Print("END sub-command\n")
	}
}
