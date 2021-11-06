package main

import(
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"log"
	"os"
)

func main() {
	var mainErr error

	defer func() {
		if mainErr != nil {
			log.Fatal(mainErr)
		}
		os.Exit(0)
	}()

	//1. Make a new AWS config
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		mainErr = fmt.Errorf("could not get AWS configuration with default methods; %v", err.Error())
		return
	}
}