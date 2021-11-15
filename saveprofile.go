package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"os"
	"os/exec"
)

// saveToLocalProfile Save the credentials to a local config file using the aws cli.
func saveToLocalProfile(creds  *iam.CreateAccessKeyOutput) error {
	awsProfile := os.Getenv("AWS_PROFILE")
	err1 := runCmd("aws", "configure", "set", "aws_access_key_id", *creds.AccessKey.AccessKeyId, "--profile", awsProfile)
	if err1 != nil {
		return err1
	}

	err2 := runCmd("aws", "configure", "set", "aws_secret_access_key", *creds.AccessKey.SecretAccessKey, "--profile", awsProfile)
	if err2 != nil {
		return err2
	}

	return nil
}

// runCmd run a command
func runCmd(program string, args ...string) error {
	cmd := exec.Command(program, args...)
	cmdOut, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return cmdErr
	}

	exitCode := cmd.ProcessState.ExitCode()
	if exitCode != 0 {
		return fmt.Errorf("exit code %v, failed to run command %v", cmdOut, exitCode)
	}

	return nil
}