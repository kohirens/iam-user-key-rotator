package main

// All flag usage/instructions/documentation goes in here.

var flagUsages = map[string]string{
	"help":           "-h, -help\n\tDisplay usage info for all arguments, flags, and subcommands.",
	"maxDaysAllowed": "[maxDaysAllowed] int\n\tAn integer representing the maximum number of days before this app will remove or rotate the IAM key/secret pair.",
	"maxKeysAllowed": "[maxKeysAllowed] int\n\tAn integer representing the maximum number of keys that should exist on an IAM user.",
	"filename":       "[filename] string\n\tPath of a file to store a new IAM key/secret pair.",
	"region":         "<region> string\n\tAn AWS region.",
}
