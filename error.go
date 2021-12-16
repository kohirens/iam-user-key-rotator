package main

var errors = struct {
	probMakingNewKey,
	regionMissing,
	translateKeyToJsonErr,
	updateCiContextErr,
	writingNewKeyErr string
}{
	regionMissing:         "the -region flag is required and must not be an empty string",
	translateKeyToJsonErr: "problem translating the new access key to JSON: %v",
	updateCiContextErr:    "failed to update context: %v",
	writingNewKeyErr:      "problem writing the new access key to a file: %v",
	probMakingNewKey:      "problem with making a new access key: %v",
}
