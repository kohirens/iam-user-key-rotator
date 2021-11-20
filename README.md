# IAM User Key Rotator

## Summary

Use an IAM user to auto rotate its own programmable key/pair every so many
(default is 30) days.

## Description

This is meant to run in a CI workflow that requires IAM credentials. It will
check its credentials are current, if not, then use its current credentials to
make new credentials; replace the existing credentials (in the CI environment).
Then delete the old/existing one from AWS.

NOTE: You are responsible for updating the existing ones with the new credentials
      placed in the environment.

This programs uses its current AWS credentials to auto rotate its own
credentials. To perform the actions needed the IAM user should have an inline
policy that like so:

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

Because you can have credentials store anywhere, we've opted to have them placed
in a JSON file. Should the CI fail, and you lose the new key, then the only way
to recover is to delete the key/secret combo, manually make a new key/secret
pair, and manually update your local credentials or CI context.

NOTE: Currently AWS only allows 2 programmatic key/secret pairs per
IAM user. Because of this restriction, the user can only have 1 secret/key pair
at a time, as the script needs 1 to be available
to make new one key for the rotation (30 days by default, see `maxDaysAllowed` flag).

It should also be allowed to work with a Lambda, to rotate a list of IAM users
after `maxDaysAllowed` days. But this functionality can also be a separate CLI from this
project.

1. Load AWS IAM Credentials.
2. Read all existing IAM user keys.
3. Calculate how many days old the keys are.
4. If only there are only `maxKeysAllowed` and they are active less than `maxDaysAllowed`, then do nothing.
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