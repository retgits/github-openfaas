// Package function handles the invocations of the function to create Trello cards.
package function

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/go-github/v25/github"
	handler "github.com/openfaas-incubator/go-function-sdk"
	"golang.org/x/oauth2"
)

// TrelloEvent contains all the details of the card to be created
type TrelloEvent struct {
	Card   Card    `json:"card"`
	Config *Config `json:"config,omitempty"`
}

// Card contains the details of the card itself
type Card struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Config contains the details of where the card should be created
type Config struct {
	Board *string `json:"board,omitempty"`
	List  *string `json:"list,omitempty"`
}

// getSecret reads a secret mounted to the container in OpenFaaS. The input is the name
// of the secret and the function returns either a byte array containing the secret or
// an error.
func getSecret(name string) (secretBytes []byte, err error) {
	// read from the openfaas secrets folder
	secretBytes, err = ioutil.ReadFile(fmt.Sprintf("/var/openfaas/secrets/%s", name))
	if err != nil {
		// read from the original location for backwards compatibility with openfaas <= 0.8.2
		secretBytes, err = ioutil.ReadFile(fmt.Sprintf("/run/secrets/%s", name))
	}

	return secretBytes, err
}

// Marshal creates a bytearray of a TrelloEvent object
func (r *TrelloEvent) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// getEnvKey tries to get the specified key from the OS environment and returns either the
// value or the fallback that was provided
func getEnvKey(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Handle a function invocation
func Handle(req handler.Request) (handler.Response, error) {
	accessToken, err := getSecret("github-accesstoken")
	if err != nil {
		return handler.Response{
			Body:       []byte(fmt.Sprintf("error reading GitHub personal access token: %s", err.Error())),
			StatusCode: http.StatusInternalServerError,
		}, err
	}

	// Create a new GitHub client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(accessToken)},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	timeInterval, err := strconv.Atoi(getEnvKey("interval", "120"))
	if err != nil {
		return handler.Response{
			Body:       []byte(fmt.Sprintf("error getting timeinterval: %s", err.Error())),
			StatusCode: http.StatusInternalServerError,
		}, err
	}

	// Create a new time to make sure we check for new issues from the previous
	// execution of this function.
	interval := time.Duration(timeInterval) * time.Minute
	t := time.Now().Add(-interval)

	issueOpts := github.IssueListOptions{Since: t}
	issues, _, err := client.Issues.List(ctx, false, &issueOpts)
	if err != nil {
		return handler.Response{
			Body:       []byte(fmt.Sprintf("error getting new issues from GitHub: %s", err.Error())),
			StatusCode: http.StatusInternalServerError,
		}, err
	}

	trelloBoard := getEnvKey("trelloboard", "Main")
	trelloList := getEnvKey("trellolist", "Tomorrow")
	url := fmt.Sprintf("%s/function/%s", getEnvKey("ofgateway", "http://gateway.openfaas:8080"), getEnvKey("trellofunction", "trellocard"))

	for idx := range issues {
		issue := issues[idx]

		event := TrelloEvent{
			Card{
				Title:       issue.GetTitle(),
				Description: "Repository: " + issue.GetRepository().GetHTMLURL() + "\nDirect link: " + issue.GetHTMLURL(),
			},
			&Config{
				Board: &trelloBoard,
				List:  &trelloList,
			},
		}

		payload, err := event.Marshal()
		if err != nil {
			return handler.Response{
				Body:       []byte(fmt.Sprintf("error marshalling GitHub issue for Trello: %s", err.Error())),
				StatusCode: http.StatusInternalServerError,
			}, err
		}

		trelloReq, err := http.NewRequest("POST", url, bytes.NewReader(payload))
		if err != nil {
			return handler.Response{
				Body:       []byte(fmt.Sprintf("error sending message to Trello function: %s", err.Error())),
				StatusCode: http.StatusInternalServerError,
			}, err
		}

		_, err = http.DefaultClient.Do(trelloReq)
		if err != nil {
			return handler.Response{
				Body:       []byte(fmt.Sprintf("received error from Trello function: %s", err.Error())),
				StatusCode: http.StatusInternalServerError,
			}, err
		}
	}

	return handler.Response{
		Body:       []byte(fmt.Sprintf("found a total of %d new issues", len(issues))),
		StatusCode: http.StatusOK,
	}, nil
}
