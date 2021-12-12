package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"io"
	"log"
	"net/http"
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
	subCmdFlags        = "RECURSIVE_TEST_FLAGS"
	dirMode            = 0700
	localStackEndpoint = "http://localstack:4566"
	localStackGood     = "{\"status\": \"running\"}"
)

func isLocalStackAvailable() bool {
	res, err := http.Get(localStackEndpoint)
	if err == nil {
		b, e := io.ReadAll(res.Body)
		if (e == nil || e == io.EOF) && string(b) == localStackGood {
			log.Println("using localstack for AWS endpoint")
			return true
		}
	}

	return false
}

func TestMain(m *testing.M) {
	// Only runs when this environment variable is set. Allows testing the main program in a sub process.
	if _, ok := os.LookupEnv(subCmdFlags); ok {
		// Adding this code to allow unit testing using localstack.
		res, err := http.Get("http://localstack:4566")
		if err == nil {
			b, e := io.ReadAll(res.Body)
			if (e == nil || e == io.EOF) && string(b) == "{\"status\": \"running\"}" {
				log.Println("using localstack for AWS endpoint")
				optFns = awsConfigOpts{
					config.WithRegion(""),
					config.WithSharedConfigProfile(""),
					config.WithEndpointResolver(aws.EndpointResolverFunc(
						// See: https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/endpoints/
						func(service, region string) (aws.Endpoint, error) {
							return aws.Endpoint{
								PartitionID:   "aws",
								URL:           "http://localstack:4566",
								SigningRegion: region,
							}, nil
						},
					)),
				}
			}
		}
		// mock HTTP client to spy and assert.
		httpComm = &mockHttpClient{0}
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
	if !isLocalStackAvailable() {
		tester.Skip("unknown environment, skipping")
	}

	// Should only run in a test environment, or it could change your real IAM keys.
	var tests = []struct {
		name     string
		wantCode int
		args     []string
	}{
		{"noFlags", 1, []string{}},
		{"withRegion", 0, []string{"-region", "us-east-2"}},
		{"withCircleSuccess", 0, []string{"-region", "us-east-2", "--circleci", "1234"}},
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

func TestGetIamKeyStats(tester *testing.T) {
	if !isLocalStackAvailable() {
		tester.Skip("unknown environment, skipping")
	}
	s1 := "ABC123"
	t1 := time.Date(2021, 12, 15, 1, 0, 0, 0, time.UTC)
	s2 := "ABCD1234"
	t2 := time.Date(2021, 12, 10, 1, 0, 0, 0, time.UTC)

	// Should only run in a test environment, or it could change your real IAM keys.
	var tests = []struct {
		name        string
		keys        []types.AccessKeyMetadata
		daysAllowed int
		currentId   string
		wantOld     int
		wantValid   int
	}{
		{"1_valid", []types.AccessKeyMetadata{{AccessKeyId: &s1, CreateDate: &t1}}, 1, s1, 0, 1},
		{"1_old", []types.AccessKeyMetadata{{AccessKeyId: &s2, CreateDate: &t2}}, 1, s2, 1, 0},
	}

	for _, test := range tests {
		tester.Run(test.name, func(t *testing.T) {
			got := getIamKeyStats(test.keys, test.daysAllowed, *test.keys[0].AccessKeyId)

			if got.current != test.currentId {
				t.Errorf("current ids do now match. want %q, got %q", test.currentId, got.current)
			}

			if len(got.old) != test.wantOld {
				t.Errorf("old array not what expected; want %q, got %q", test.wantOld, len(got.old))
			}

			if len(got.valid) != test.wantValid {
				t.Errorf("valid array not what expected; want %q, got %q", test.wantValid, len(got.valid))
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
