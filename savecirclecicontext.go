package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"io/ioutil"
	"net/http"
	"strings"
)

type httpCommunicator interface {
	Do(req *http.Request) (*http.Response, error)
}

func updateCircleCIContextVar(name, val, token string, client httpCommunicator) error {
	url := "https://circleci.com/api/v2/context/%7Bcontext-id%7D/environment-variable/" + name

	payload := strings.NewReader("{\"value\":\"" + val + "\"}")

	req, _ := http.NewRequest("PUT", url, payload)

	req.Header.Add("content-type", "application/json")
	req.Header.Add("authorization", "Basic "+token)

	res, _ := client.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode != 200 {
		return fmt.Errorf(errors.updateCiContextErr, string(body))
	}

	return nil
}

// saveToCircleContext
func saveToCircleContext(creds *iam.CreateAccessKeyOutput, cciToken string, hc httpCommunicator) error {
	if err := updateCircleCIContextVar(keyVarName, *creds.AccessKey.AccessKeyId, cciToken, hc); err != nil {
		return err
	}

	if err := updateCircleCIContextVar(secretVarName, *creds.AccessKey.SecretAccessKey, cciToken, hc); err != nil {
		return err
	}

	return nil
}
