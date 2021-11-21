package main

import (
	"flag"
	"fmt"
)

// This is the struct that defines all of application flags.
type applicationFlags struct {
	maxDaysAllowed *int
	maxKeysAllowed *int
	region,
	filename,
	profile *string
}

// appFlags Is the implementation of the applicationFlags type which you use at runtime.
var appFlags = new(applicationFlags)

// defineFlags Define all application flags.
func (af *applicationFlags) define() {
	appFlags.maxDaysAllowed = flag.Int("maxDaysAllowed", 30, flagUsages["maxDaysAllowed"])
	appFlags.maxKeysAllowed = flag.Int("maxKeysAllowed", 1, flagUsages["maxKeysAllowed"])
	appFlags.region = flag.String("region", "", flagUsages["region"])
	appFlags.filename = flag.String("filename", "new-aws-access-key.json", flagUsages["filename"])
	appFlags.profile = flag.String("profile", "", flagUsages["profile"])
}

// check Verify that all flags are set appropriately.
func (af *applicationFlags) check() error {
	if *(af.region) == "" {
		return fmt.Errorf(errors.regionMissing)
	}

	return nil
}
