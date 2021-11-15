package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type awsKeyPair struct {
	Id string`json:"aws_access_key_id" csv:"User Name"`
	Key string`json:"aws_secret_access_key" csv:"Access key ID"`
	Username string`json:"username" csv:"Secret access key"`
}

func main() {
	var mainErr error

	var validKeys []types.AccessKeyMetadata
	maxDaysAllowed := 30
	var deleteKeys []types.AccessKeyMetadata
	maxKeysAllowed := 1
	region := "us-east-2"
	filename := "new-aws-access-key.json"

	defer func() {
		if mainErr != nil {
			log.Fatal(mainErr)
		}
		os.Exit(0)
	}()

	// 1. Make a new AWS config to load the Shared AWS Configuration (such as ~/.aws/config)
	awsConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		mainErr = fmt.Errorf("could not get AWS configuration with default methods; %v", err.Error())
		return
	}

	// get current access key id:

	creds, err6 := awsConfig.Credentials.Retrieve(context.TODO())
	if err6 != nil {
		mainErr = fmt.Errorf("could not get current AWS key ID; %v", err.Error())
	}

	currentId := creds.AccessKeyID

	// 1.5 New IAM client
	iamClient := iam.NewFromConfig(awsConfig)

	// 2. Load AWS IAM Credentials.
	liko, err2 := iamClient.ListAccessKeys(context.TODO(), &iam.ListAccessKeysInput{})
	if err2 != nil {
		mainErr = err2
		return
	}

	numKeys := len(liko.AccessKeyMetadata)

	// 3. Read IAM user keys.
	for i, v := range liko.AccessKeyMetadata {
		// Calculate how many days old the key is.
		daysOld := DaysOld(v.CreateDate)
		log.Printf("%d. id: %s, status: %v, username: %s, date: %v, days: %v\n", i+1, *v.AccessKeyId, v.Status, *v.UserName, v.CreateDate, daysOld)

		//5. If older than maxDaysAllowed, then rotate the key.
		if daysOld > maxDaysAllowed && numKeys > maxKeysAllowed {
			log.Printf("will delete key: %v", *v.AccessKeyId)
			deleteKeys = append(deleteKeys, v)
			continue
		}

		// 5. If less than 30 days, then do nothing.
		validKeys = append(validKeys, v)
	}

	log.Printf("number of keys %v", numKeys)
	log.Printf("\t%v are valid keys", len(validKeys))
	log.Printf("\t%v will be removed", len(deleteKeys))

	// make sure there is room to make a new key.
	for _, v := range deleteKeys {
		// delete all keys marked for deletion, except the one we are using.
		if *v.AccessKeyId != currentId {
			daki := &iam.DeleteAccessKeyInput{AccessKeyId: v.AccessKeyId}
			_, err7 := iamClient.DeleteAccessKey(context.TODO(), daki)
			if err7 != nil {
				mainErr = fmt.Errorf("could not delete key %q; %v", *v.AccessKeyId, err7.Error())
			}
			log.Printf("removed key %v\n", *v.AccessKeyId)
		}
	}

	numValidKeys := len(validKeys)
	if numValidKeys > maxKeysAllowed {
		// delete keys that we are not using, until we get to the max allowed.
		for _, v := range validKeys {
			if *v.AccessKeyId != currentId {
				daki := &iam.DeleteAccessKeyInput{AccessKeyId: v.AccessKeyId}
				_, err7 := iamClient.DeleteAccessKey(context.TODO(), daki)
				if err7 != nil {
					mainErr = fmt.Errorf("could not delete key %q; %v", *v.AccessKeyId, err7.Error())
				}
				log.Printf("removed key %v\n", *v.AccessKeyId)
				numValidKeys--
				if numValidKeys <= maxKeysAllowed {
					break
				}
			}
		}
	}

	// If no valid keys are left, then make a new one.
	if numValidKeys == 0 {
		log.Println("no valid keys, making a new key")
		newKey, err3 := iamClient.CreateAccessKey(context.TODO(), &iam.CreateAccessKeyInput{})
		if err3 != nil {
			mainErr = fmt.Errorf("problem with making a new access key: %v", err3.Error())
		}

		nk := awsKeyPair{ *newKey.AccessKey.AccessKeyId, *newKey.AccessKey.SecretAccessKey, *newKey.AccessKey.UserName}

		content, errj := json.Marshal(nk)
		if errj != nil {
			mainErr = fmt.Errorf("problem writing new access key %v", err.Error())
			return
		}

		err4 := ioutil.WriteFile(filename, []byte(content), 0774)
		if err4 != nil {
			mainErr = fmt.Errorf("problem writing new access key %v", err.Error())
			return
		}

		saveToLocalProfile(newKey)
		log.Println("new key saved to local profile")
	}

	// Delete any remaining keys (which should only be the current key if any).
	for _, v := range deleteKeys {
		daki := &iam.DeleteAccessKeyInput{AccessKeyId: v.AccessKeyId}
		_, err7 := iamClient.DeleteAccessKey(context.TODO(), daki)
		if err7 != nil {
			mainErr = fmt.Errorf("could not delete key %q; %v", *v.AccessKeyId, err7.Error())
		}
		log.Printf("removed key %v\n", *v.AccessKeyId)
	}
}

// DaysOld Calculate the days passed since the date.
func DaysOld(someDate *time.Time) int {
	days := time.Now().Sub(*someDate).Hours() / 24

	return int(days)
}
