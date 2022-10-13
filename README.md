# IAM User Key Rotator

## Summary

Have your IAM user automatically rotate its own programmable key/pair every so many
(default is 30) days.

## Description

This is meant to run in a CI workflow that requires IAM credentials. It will
check its credentials are current, if not, then use its current credentials to
make new credentials; replace the existing credentials (in the CI environment, using the API).
Then delete the old/existing one from AWS.

## Status

[![CircleCI](https://circleci.com/gh/kohirens/version-release-orb/tree/main.svg?style=svg)](https://circleci.com/gh/kohirens/version-release-orb/tree/main)

### Use Cases

* Run to rotate your local keys.
* In a Circle CI job for keys stored in a context.
* Anywhere you can run this tool.

This programs uses currently set AWS config/credentials to auto rotate the current IAM user on
credentials. To perform the actions needed, then the IAM user MUST have a
policy that allows these actions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "iam:DeleteAccessKey",

                "iam:CreateAccessKey",
                "iam:ListAccessKeys"
            ],
            "Resource": "<ARN of the IAM user itself>"
        }
    ]
}
```

Once a new key/pair is made, it will be placed
in a JSON file where the command was run. Should it fail, and you lose the new key, then you need to
go into the AWS console and make a new key pair and manually update the environment.

NOTE: Currently AWS only allows 2 programmatic key/secret pairs per
IAM user. Because of this restriction, the user can only have 1 secret/key pair
at a time, as the script needs 1 to be available
to make a new key for the rotation (30 days by default, see `maxDaysAllowed` flag).

In theory you could run this in a Lambda to rotate a list of IAM users
after `maxDaysAllowed` day, then update then in a secrets manager through API calls.
But this functionality would need to be built out as a separate CLI from this
project.

## How It Works

Here are the steps this program takes:

1. Load AWS IAM credentials (pulled from AWS defaults).
2. Read all existing keys for the user.
3. Calculate how many days old the keys are.
    1. If they are and they are active less than `maxDaysAllowed`, then do nothing.
5. If any keys are older than the `maxKeysAllowed` days:
   1. Remove all except `maxKeysAllowed`, then:
      1. rotate remaining key
      2. and update current user key storage.

## Set AWS Profile with an Environment Variable

Set a variable at the shell level (will work until you close the terminal):

```shell
Env:AWS_PROFILE="someProfile"
```

Set a variable for the Windows user account level (and not the system level):

```shell
[Environment]::SetEnvironmentVariable("AWS_PROFILE","someProfile","User")
```
