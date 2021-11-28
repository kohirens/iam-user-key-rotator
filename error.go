package main

var errors = struct {
	regionMissing,
	updateCiContextErr string
}{
	regionMissing:      "the -region flag is required and must not be an empty string",
	updateCiContextErr: "failed to update context: %v",
}
