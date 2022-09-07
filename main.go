package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// AWSEd API formatted correctly
var awsedApi = fmt.Sprintf("AWSEd api_key=%v", os.Getenv("AWSED_API_KEY"))

// TODO: check that it's not null in main
var config, _ = LoadConfig("config.json")

// Configurations:
// awsed API key [x]
// list of persistent volumes to delete
// {user}-dsmlp-datasets, {user}-dsmlp-datasets-nfs

type Config struct {
	ApiUrl string `json:"api_url"`
}

/*
Loads and decods json configurations into a go struct

Params:

- filname an absolute or relative path to config.json

Returns:

- config structure

- error if file not found
*/
func LoadConfig(filename string) (Config, error) {
	var config Config
	configFile, err := os.Open(filename)

	if err != nil {
		return config, err
	}

	defer configFile.Close()

	// Creats a new json decoder and decode contents into config var
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config, err
}

type AWSInterface interface {
	GetEnrollments()
}

type AWSed struct {
}

// We just use usernames, since we don't need Uids or first names
type dsmlpUser struct {
	Username string `json:"username`
}

/*
Gets a list of active enrolled users using dslmp

Returns:

- []dsmlpUser - a list of structures storing active user's usernames

- error
*/
func GetEnrollments() ([]dsmlpUser, error) {

	var dsmlpUsers []dsmlpUser
	reqUrl := config.ApiUrl + "/enrollments?env=dsmlp"

	// Create a template for a standard GET request for all active enrolled users,
	// that use dsmlp
	request, err := http.NewRequest(
		http.MethodGet,
		reqUrl,
		nil,
	)

	// Will never pop up, but compiler requires it to handle this err
	if err != nil {
		return dsmlpUsers, err
	}

	// Add API key for header
	request.Header.Add("Authorization", awsedApi)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return dsmlpUsers, fmt.Errorf("error reading HTTP response body: %v", err)
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return dsmlpUsers, fmt.Errorf("error reading HTTP response body: %v", err)
	}

	json.Unmarshal(responseBytes, &dsmlpUsers)

	return dsmlpUsers, err

}

// TODO
type MockAWSed struct {
}

type K8sInterface interface {
	GetNamespace()
	DeleateNamespace()
	DeleateNamespacePV()
}

type K8s struct {
}

type MockK8s struct {
}

type ControllerInterface interface {
	CleanupListPV()

	// private helper functions
	GetActiveK8sUsers()
	DiffActive()
}

type Controller struct {
}

type DryRunController struct {
}

func main() {
	result, _ := GetEnrollments()
	for index := range result {
		fmt.Println(result[index].Username)
	}

}
