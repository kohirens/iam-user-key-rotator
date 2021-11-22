package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	fixturesDir = "testdata"
	testTmp     = "tmp"
	// subCmdFlags space separated list of command line flags.
	subCmdFlags = "RECURSIVE_TEST_FLAGS"
	dirMode     = 0700
)

func TestMain(m *testing.M) {
	// Only runs when this environment variable is set.
	if _, ok := os.LookupEnv(subCmdFlags); ok {
		// See: https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/endpoints/
		localStackResolver = aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           "http://localstack:4566",
				SigningRegion: region,
			}, nil
		})
		runAppMain()
	}

	// delete all tmp files before running all test, but leave them afterward for manual inspection.
	_ = os.RemoveAll(testTmp)
	// Set up a temporary dir for generate files
	_ = os.Mkdir(testTmp, dirMode) // set up a temporary dir for generate files

	// Run all tests
	exitCode := m.Run()
	// Clean up
	os.Exit(exitCode)
}

func TestDaysOld(tester *testing.T) {
	cases := []struct {
		name string
		fix  time.Time
	}{
		{"10days", time.Now().AddDate(0, 0, -10)},
	}

	for _, test := range cases {
		tester.Run(test.name, func(t *testing.T) {
			got := DaysOld(&test.fix)
			if got != 10 {
				t.Errorf("want %v, got %v", test.fix, got)
			}
		})
	}
}

func TestFlags(tester *testing.T) {
	var tests = []struct {
		name     string
		wantCode int
		args     []string
	}{
		{"noFlags", 1, []string{}},
		{"withRegion", 0, []string{"-region", "us-east-2"}},
	}

	for _, test := range tests {
		tester.Run(test.name, func(t *testing.T) {
			cmd := getTestBinCmd(test.args)

			cmdOut, cmdErr := cmd.CombinedOutput()

			got := cmd.ProcessState.ExitCode()

			// Debug
			showCmdOutput(cmdOut, cmdErr)

			if got != test.wantCode {
				t.Errorf("unexpected error on exit. want %q, got %q", test.wantCode, got)
			}
		})
	}
}

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
