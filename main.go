package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type awsKeyPair struct {
	Id       string `json:"aws_access_key_id" csv:"User Name"`
	Key      string `json:"aws_secret_access_key" csv:"Access key ID"`
	Username string `json:"username" csv:"Secret access key"`
}

func init() {
	appFlags.define()
}

var localStackResolver aws.EndpointResolverFunc
var hClient httpCommunicator

func main() {
	var mainErr error
	var validKeys []types.AccessKeyMetadata
	var deleteKeys []types.AccessKeyMetadata

	defer func() {
		if mainErr != nil {
			log.Fatal(mainErr)
		}

		log.Println("exiting with code 0")
		os.Exit(0)
	}()

	flag.Parse()

	if err := appFlags.check(); err != nil {
		mainErr = err
		return
	}

	maxDaysAllowed := *appFlags.maxDaysAllowed
	maxKeysAllowed := *appFlags.maxKeysAllowed
	region := *appFlags.region
	filename := *appFlags.filename
	profile := *appFlags.profile
	cciToken := *appFlags.circleci

	// Make a new AWS config to load the Shared AWS Configuration (such as ~/.aws/config).
	var awsConfig aws.Config
	var err0 error

	if localStackResolver != nil {
		// TODO: : Find a better way to allow this to be tested.
		// Adding this code to allows unit testing using localstack.
		log.Println("using localstack")
		awsConfig, err0 = config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile), config.WithEndpointResolver(localStackResolver))
	} else {
		awsConfig, err0 = config.LoadDefaultConfig(context.TODO(), config.WithRegion(region), config.WithSharedConfigProfile(profile))
	}
	if err0 != nil {
		mainErr = fmt.Errorf("could not get AWS configuration with default methods; %v", err0.Error())
		return
	}

	// Get current access key id.
	creds, err6 := awsConfig.Credentials.Retrieve(context.TODO())
	if err6 != nil {
		mainErr = fmt.Errorf("could not get current AWS key ID; %v", err6.Error())
		return
	}

	currentId := creds.AccessKeyID

	log.Printf("Region/IAM user : %v/%v\n", awsConfig.Region, currentId)
	// Init a new IAM client.
	iamClient := iam.NewFromConfig(awsConfig)

	// Load AWS IAM Credentials.
	liko, err2 := iamClient.ListAccessKeys(context.TODO(), &iam.ListAccessKeysInput{})
	if err2 != nil {
		mainErr = err2
		return
	}

	numKeys := len(liko.AccessKeyMetadata)

	// Read IAM user keys.
	// TODO: refactor as func getIamKeys
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
	if errX := makeRoomForKey(currentId, deleteKeys, iamClient); errX != nil {
		mainErr = errX
		return
	}

	// TODO: Extract as func validKeys
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

		nk := awsKeyPair{*newKey.AccessKey.AccessKeyId, *newKey.AccessKey.SecretAccessKey, *newKey.AccessKey.UserName}

		content, errj := json.Marshal(nk)
		if errj != nil {
			mainErr = fmt.Errorf("problem writing new access key %v", err0.Error())
			return
		}

		err4 := ioutil.WriteFile(filename, []byte(content), 0774)
		if err4 != nil {
			mainErr = fmt.Errorf("problem writing new access key %v", err0.Error())
			return
		}

		if hClient == nil {
			hClient = &http.Client{}
		}

		saveMode := ""
		if *(appFlags.circleci) != "" {
			saveMode = "circleci"
		}

		switch saveMode {
		case "circleci":
			if err := updateCircleCIContextVar("AWS_ACCESS_KEY_ID", *newKey.AccessKey.AccessKeyId, cciToken, hClient) ; err != nil {
				mainErr = err
				return
			}
			if err := updateCircleCIContextVar("AWS_SECRET_ACCESS_KEY", *newKey.AccessKey.SecretAccessKey, cciToken, hClient) ; err != nil {
				mainErr = err
				return
			}
		default:
			saveErr := saveToLocalProfile(newKey)
			if saveErr != nil {
				mainErr = saveErr
				return
			}
		}


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

func deleteKey(deleteKeys []types.AccessKeyMetadata, iamClient *iam.Client) error {
	for _, v := range deleteKeys {
		daki := &iam.DeleteAccessKeyInput{AccessKeyId: v.AccessKeyId}
		_, err7 := iamClient.DeleteAccessKey(context.TODO(), daki)
		if err7 != nil {
			return fmt.Errorf("could not delete key %q; %v", *v.AccessKeyId, err7.Error())
		}
		log.Printf("removed key %v\n", *v.AccessKeyId)
	}

	return nil
}

// makeRoomForKey Deletes all IAM keys in the delete key list except for the current access ID in use.
func makeRoomForKey(currentId string, deleteKeys []types.AccessKeyMetadata, iamClient *iam.Client) error {
	for _, v := range deleteKeys {
		// delete all keys marked for deletion, except the one we are using.
		if *v.AccessKeyId != currentId {
			daki := &iam.DeleteAccessKeyInput{AccessKeyId: v.AccessKeyId}
			_, err7 := iamClient.DeleteAccessKey(context.TODO(), daki)
			if err7 != nil {
				return fmt.Errorf("could not delete key %q; %v", *v.AccessKeyId, err7.Error())
			}
			log.Printf("removed key %v\n", *v.AccessKeyId)
		}
	}

	return nil
}
