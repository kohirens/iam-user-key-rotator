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

const keyVarName = "AWS_ACCESS_KEY_ID"
const secretVarName = "AWS_SECRET_ACCESS_KEY"

type awsKeyPair struct {
	Id       string `json:"aws_access_key_id" csv:"User Name"`
	Key      string `json:"aws_secret_access_key" csv:"Access key ID"`
	Username string `json:"username" csv:"Secret access key"`
}

type iamStats struct {
	current    string
	keys       []iamKeyInfo
	old, valid []*iamKeyInfo
}

type iamKeyInfo struct {
	*types.AccessKeyMetadata
	Days    int
	Expired bool
}

// awsConfigOpts shorthand to set an array of config.LoadOptionsFunc to override defaults
type awsConfigOpts []func(*config.LoadOptions) error

var httpComm httpCommunicator
var optFns []func(*config.LoadOptions) error

// RemoveKeyByIndex Removed IAM key from stats.
func (is *iamStats) RemoveKeyByIndex(idx int) {
	tmpK := is.keys[idx]
	is.keys = append(is.keys[:idx], is.keys[idx+1:]...)

	// Remove reference in old array.
	for i, v := range is.old {
		if v == &tmpK {
			is.old = append(is.old[:i], is.old[i+1:]...)
		}
	}
	// Remove reference in valid array.
	for i, v := range is.valid {
		if v == &tmpK {
			is.valid = append(is.valid[:i], is.valid[i+1:]...)
		}
	}
}

// IsCurrentKeyExpired Indicates the current key is less than maxDaysOld.
func (is *iamStats) IsCurrentKeyExpired() bool {
	for _, v := range is.keys {
		if *v.AccessKeyId == is.current {
			return v.Expired
		}
	}

	// The current key should always be present, unless there is broken logic that deleted it before it was time.
	// Do NOT call this after the current key has been replaced. Which happens when the key has expired,
	// and it must be replaced. Some lines later in main function.
	panic("could not find current key")
}

func init() {
	appFlags.define()
}

func main() {
	var mainErr error

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
	filename := *appFlags.filename

	// Make a new AWS config to load the Shared AWS Configuration (such as ~/.aws/config).
	awsConfig, err0 := getAwsConfig(appFlags)
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

	// Init a new IAM client.
	iamClient := iam.NewFromConfig(awsConfig)

	// Query IAM for any keys.
	liko, err2 := iamClient.ListAccessKeys(context.TODO(), &iam.ListAccessKeysInput{})
	if err2 != nil {
		mainErr = err2
		return
	}

	// Determine which keys are older than days allowed
	iamKeyStats := getIamKeyStats(liko.AccessKeyMetadata, maxDaysAllowed, creds.AccessKeyID)
	displayIamStats(iamKeyStats)

	// make sure there is room to make a new key.
	if errX := makeRoomForKey(currentId, iamKeyStats.old, iamClient); errX != nil {
		mainErr = errX
		return
	}

	if errX := removeExcessKeys(iamKeyStats, maxKeysAllowed, currentId, iamClient); errX != nil {
		mainErr = errX
		return
	}

	// Make a new key when the current one has expired.
	if iamKeyStats.IsCurrentKeyExpired() {
		log.Println("no valid keys, making a new key")

		newKey, errX := makeNewKey(iamKeyStats, iamClient)
		if errX != nil {
			mainErr = errX
			return
		}

		if httpComm == nil {
			httpComm = &http.Client{}
		}

		if err := save(newKey, appFlags, httpComm, filename); err != nil {
			mainErr = err
			return
		}
	}

	// Delete any remaining keys (which should only be the current key if any).
	for _, v := range iamKeyStats.old {
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

// getAwsConfig Get an AWS Config, with optional overrides.
func getAwsConfig(ac *applicationFlags) (aws.Config, error) {
	if optFns == nil {
		optFns = awsConfigOpts{
			config.WithRegion(*ac.region),
			config.WithSharedConfigProfile(*ac.profile),
		}
	}

	return config.LoadDefaultConfig(context.TODO(), optFns...)
}

// makeRoomForKey Deletes all IAM keys in the delete key list except for the current access ID in use.
func makeRoomForKey(currentId string, deleteKeys []*iamKeyInfo, iamClient *iam.Client) error {
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

// makeNewKey Add a new IAM key.
func makeNewKey(stats *iamStats, iamClient *iam.Client) (*iam.CreateAccessKeyOutput, error) {

	// If no valid keys are left, then make a new one.
	if stats.IsCurrentKeyExpired() {
		return nil, nil
	}

	log.Println(stdMsgs.expireKey)

	newKey, err1 := iamClient.CreateAccessKey(context.TODO(), &iam.CreateAccessKeyInput{})
	if err1 != nil {
		return nil, fmt.Errorf(errors.probMakingNewKey, err1.Error())
	}

	return newKey, nil
}

// save AWS credentials to a medium.
func save(creds *iam.CreateAccessKeyOutput, ac *applicationFlags, hc httpCommunicator, filename string) error {
	saveMode := ""
	if *(appFlags.circleci) != "" {
		saveMode = "circleci"
	}

	// Always save to a local file.
	if err := saveToFile(creds, filename); err != nil {
		return err
	}

	switch saveMode {
	case "circleci":
		log.Println("saving to Circle CI context")
		return saveToCircleContext(creds, *ac.circleci, hc)
	default:
		log.Println("saving to local credentials/profile")
		return saveToLocalProfile(creds)
	}
}

// saveToFile Save the new key to a local file as JSON.
func saveToFile(newKey *iam.CreateAccessKeyOutput, filename string) error {
	nk := awsKeyPair{*newKey.AccessKey.AccessKeyId, *newKey.AccessKey.SecretAccessKey, *newKey.AccessKey.UserName}

	content, err1 := json.Marshal(nk)
	if err1 != nil {
		return fmt.Errorf(errors.translateKeyToJsonErr, err1.Error())
	}

	if err := ioutil.WriteFile(filename, content, 0774); err != nil {
		return fmt.Errorf(errors.writingNewKeyErr, err.Error())
	}

	return nil
}

func newIamStats(c string) *iamStats {
	stats := &iamStats{
		current: c,
		keys:    make([]iamKeyInfo, 0),
		old:     make([]*iamKeyInfo, 0),
		valid:   make([]*iamKeyInfo, 0),
	}
	return stats
}

func getIamKeyStats(ak []types.AccessKeyMetadata, daysAllowed int, currentId string) *iamStats {
	stats := newIamStats(currentId)

	for _, v := range ak {
		daysOld := DaysOld(v.CreateDate)
		k := iamKeyInfo{
			&v,
			daysOld,
			daysOld > daysAllowed,
		}
		stats.keys = append(stats.keys, k)

		// When older than daysAllowed, then mark for deletion.
		if daysOld > daysAllowed {
			stats.old = append(stats.old, &k)
			continue
		}

		// otherwise, its valid.
		stats.valid = append(stats.old, &k)
	}

	return stats
}

// displayIamStats Display info that allows the user to understand what is happening.
func displayIamStats(stats *iamStats) {
	// Header
	log.Println("key id               | status | username | days old | date")

	for _, v := range stats.keys {
		// Calculate how many days old the key is.
		daysOld := DaysOld(v.CreateDate)
		log.Printf("%s | %v | %s | %v | %v\n", *v.AccessKeyId, v.Status, *v.UserName, daysOld, v.CreateDate)
	}

	log.Printf("number of keys %v", len(stats.keys))
	log.Printf("\t%v are valid keys", len(stats.valid))
	log.Printf("\t%v will be removed", len(stats.old))
}

func removeExcessKeys(stats *iamStats, maxKeysAllowed int, currentId string, iamClient *iam.Client) error {
	numKeys := len(stats.keys)

	if numKeys <= maxKeysAllowed {
		return nil
	}
	// delete keys that we are not using, until we get to the max allowed.
	for i, v := range stats.keys {
		if *v.AccessKeyId == currentId {
			continue
		}
		if v.Expired || len(stats.keys) > maxKeysAllowed {
			daki := &iam.DeleteAccessKeyInput{AccessKeyId: v.AccessKeyId}
			_, err7 := iamClient.DeleteAccessKey(context.TODO(), daki)
			if err7 != nil {
				return fmt.Errorf("could not delete key %q; %v", *v.AccessKeyId, err7.Error())
			}
			// Remove any reference to the deleted key.
			stats.RemoveKeyByIndex(i)

			log.Printf("removed key %v\n", *v.AccessKeyId)
			numKeys--
			if numKeys <= maxKeysAllowed {
				break
			}
		}
	}

	return nil
}
