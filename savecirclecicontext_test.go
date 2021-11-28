package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

type mockHttpClient struct {
	ResponseType int
}

func (mhc *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	switch mhc.ResponseType {
	case 1:
		return &http.Response{StatusCode: 400, Body: ioutil.NopCloser(strings.NewReader("err"))}, nil
	default:
		return &http.Response{StatusCode: 200, Body: ioutil.NopCloser(strings.NewReader(""))}, nil
	}
}

func TestUpdateCircleCIContextVar(tester *testing.T) {
	cases := []struct {
		name   string
		want   error
		client httpCommunicator
	}{
		{"updateFails", fmt.Errorf(errors.updateCiContextErr, "err"), &mockHttpClient{1}},
		{"updateSucceeds", nil, &mockHttpClient{0}},
	}

	for _, test := range cases {
		tester.Run(test.name, func(t *testing.T) {
			got := updateCircleCIContextVar("", "", "", test.client)
			// Had to extract the error messages a compare them.
			// Handle nil case separately
			if (got != nil && got.Error() != test.want.Error()) || (got == nil && got != test.want) {
				t.Errorf("want %v, got %v", test.want, got)
				return
			}
		})
	}
}
