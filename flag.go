package main

import (
	"flag"
	"fmt"
)

// This is the struct that defines all application flags.
type applicationFlags struct {
	maxDaysAllowed,
	maxKeysAllowed *int
	circleci,
	region,
	filename,
	profile *string
}

// appFlags Is what you use at runtime, it is the implementation of the applicationFlags type.
var appFlags = new(applicationFlags)

// defineFlags Define all application flags.
func (af *applicationFlags) define() {
	// NOTE: This code is redundant, but if we try to dry it out then it could get overly complicated and ruin the
	// simplicity. Though I do like the idea of only adding a new field to the applicationFlags and automating lines
	// added here and elsewhere.
	appFlags.maxDaysAllowed = flag.Int("maxDaysAllowed", 30, flagUsages["maxDaysAllowed"])
	appFlags.maxKeysAllowed = flag.Int("maxKeysAllowed", 1, flagUsages["maxKeysAllowed"])
	appFlags.region = flag.String("region", "", flagUsages["region"])
	appFlags.filename = flag.String("filename", "new-aws-access-key.json", flagUsages["filename"])
	appFlags.profile = flag.String("profile", "", flagUsages["profile"])
	appFlags.circleci = flag.String("circleci", "", flagUsages["circleci"])
}

// check Verify that all flags are set appropriately.
func (af *applicationFlags) check() error {
	if *(af.region) == "" {
		return fmt.Errorf(errors.regionMissing)
	}

	return nil
}
