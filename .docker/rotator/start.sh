#!/bin/sh

# We'll use this script to manage starting and stopping this container gracefully.
# It only takes up about 00.01 CPU % allotted to the container, you can verify
# by running `docker stats` after you start a container that uses this as
# as the CMD.

set -e

shutd () {
    printf "Shutting down the container gracefully..."
    # You can run clean commands here!
    printf "done\n"
}

trap 'shutd' SIGTERM

echo "Starting up..."

end=0

printf "waiting for a file to show up"
while [ $end -lt 10 ]; do
    printf "."
    end=$((end+1))
    sleep 1
    :
done
echo "done"

iamCredsFile="/home/app/src/github.com/kohirens/iam-user-key-rotator/testdata/access-key-secret-auto-roto.json"

if [ -f "${iamCredsFile}" ]; then
  printf "Generating an AWS credentials file from localstack IAM credentials..."
  id=$(cat ${iamCredsFile} | jq -r .AccessKey.AccessKeyId)
  secret=`cat ${iamCredsFile} | jq -r .AccessKey.SecretAccessKey`

  mkdir -p ~/.aws/
  echo "[default]" > ~/.aws/config
  echo "aws_access_key_id = ${id}" >> ~/.aws/config
  echo "aws_secret_access_key = ${secret}" >> ~/.aws/config
  echo "done"
fi

# Run non-blocking commands here
go mod tidy
go mod vendor

echo "Ready!"

# This keeps the container running until it receives a signal to be stopped.
# Also very low CPU usage.
while :; do :; done & kill -STOP $! && wait $!
