# IAM User Key Rotator

## Summary

Use an IAM user to auto rotate its own programmable key/pair every so many
(default is 30) days.

## Description

This is meant to run in a CI workflow that requires IAM credentials. It will
perform the check every time, and use its current credentials to make new
credentials; replace the existing credentials (in the CI environment where they
are stored). Then delete the old/existing one from AWS.

So it is an IAM user that auto rotates its own credentials. With only 1 hacky way
of being able to retrieve them after the initial credentials have been removed
after 30 days. If you cannot figure out what I mean by "1 hacky way", then as
a last resort you can use the AWS web console to delete the existing key/secret
combo, manually make a new key/secret pair, and manually update your CI context.

NOTE: If it's not obvious, AWS only allows to programmatic key/secret pairs per
IAM user. Because of this restriction, the user can only have 1 secret/key pair
at a time, as the script needs 1 to be available
to make new ones every so many days (30 by default).

It should also be allowed to work with a Lambda, to rotate a list of IAM users
after 30 days. But this functionality can also be a separate CLI from this
project.

2. Load AWS IAM Credentials.
3. Read IAM user keys.
4. Calculate how many days old the key is
5. If older than 30 days, then rotate the key
6. If less than 30 days, then do nothing.