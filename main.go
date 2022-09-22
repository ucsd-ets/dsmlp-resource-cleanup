package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AWSEd API formatted correctly
var awsedApi = fmt.Sprintf("AWSEd api_key=%v", os.Getenv("AWSED_API_KEY"))

var config, _ = loadConfig("config.json")

type Config struct {
	ActiveUsers string   `json:"active_users_url"`
	UserUrl     string   `json:"user_url"`
	Volumes     []string `json:"volume_extensions"`
}

/*
Loads and decods json configurations into a go struct

Params:

- filname string - an absolute or relative path to config.json

Returns:

- config structure

- error if file not found
*/
func loadConfig(filename string) (Config, error) {
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

type AWSedInterface interface {
	getActiveUsers() ([]string, error)
}
type AWSed struct {
}

// We just use usernames, since we don't need Uids or first names
type AWSUser struct {
	Username string `json:"username`
}

/*
Gets a list of names of active enrolled users using dslmp

Params:

- a AWSInterface - an instanse of AWS client

Returns:

- []string - a list of active user's names

- error
*/
func (a AWSed) getActiveUsers() ([]string, error) {
	var activeUserNames []string
	var activeUsers []AWSUser
	reqUrl := config.ActiveUsers

	// Create a template for a standard GET request for all active enrolled users,
	// that use dsmlp
	request, err := http.NewRequest(
		http.MethodGet,
		reqUrl,
		nil,
	)

	// Will never pop up, but compiler requires it to handle this err
	if err != nil {
		return activeUserNames, fmt.Errorf("error reading HTTP response body: %v", err)
	}

	// Add API key for header
	request.Header.Add("Authorization", awsedApi)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return activeUserNames, fmt.Errorf("error reading HTTP response body: %v", err)
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return activeUserNames, fmt.Errorf("error reading HTTP response body: %v", err)
	}

	json.Unmarshal(responseBytes, &activeUsers)

	for _, user := range activeUsers {
		activeUserNames = append(activeUserNames, user.Username)
	}

	return activeUserNames, err

}

func getUserDetail(name string) (int, error) {

	reqUrl := config.UserUrl
	request, err := http.NewRequest(
		http.MethodGet,
		reqUrl,
		nil,
	)

	if err != nil {
		return -1, err
	}

	// Add API key for header
	request.Header.Add("Authorization", awsedApi)

	check, err := http.DefaultClient.Do(request)

	return check.StatusCode, err
}

type MockAWSed struct {
}

func (m MockAWSed) getEnrollments() ([]string, error) {
	var activeUsersNames []string
	var activeUsers []AWSUser
	userFile, err := os.Open("tests/mock_AWS.json")

	if err != nil {
		return activeUsersNames, err
	}

	defer userFile.Close()

	responseBytes, err := ioutil.ReadAll(userFile)
	if err != nil {
		return activeUsersNames, fmt.Errorf("error reading HTTP response body: %v", err)
	}

	json.Unmarshal(responseBytes, &activeUsers)

	for _, user := range activeUsers {
		activeUsersNames = append(activeUsersNames, user.Username)
	}

	return activeUsersNames, err
}

type K8sInterface interface {
	clientSetup()
	listNamespaces()
	deleateNamespace(namespace string)
	deletePV(namespace string)
	listDeleatedPV()
}

type K8s struct {
	clientset kubernetes.Interface
}

/*
Creates a valid clientset to work with k8s cluster and assigns it to K8s client

Params:

- k8s K8s - an instance of k8s client
*/
func clientSetup(k8s K8s) (kubernetes.Interface, error) {

	config, err := rest.InClusterConfig()

	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	return clientset, err
}

/*
Creates a list of names of actives namespaces

Params:

- k8s K8s - an instance of k8s client

Returns:

- []string - a list of all active namespaces in cluster
*/
func listNamespaces(k8s K8s) ([]string, error) {

	var dslmpNamespacelist []string

	namspaceList, err := k8s.clientset.CoreV1().
		Namespaces().
		List(context.Background(), v1.ListOptions{})

	if err != nil {
		return nil, err
	}

	for _, n := range namspaceList.Items {
		dslmpNamespacelist = append(dslmpNamespacelist, n.Name)
	}

	return dslmpNamespacelist, err
}

/*
Deletes a namespace by name

Params:

- k8s K8s - an instance of k8s client

- namespace string - name of a namespace that is deleated
*/
func deleteNamespace(k8s K8s, namespace string) error {
	err := k8s.clientset.CoreV1().
		Namespaces().
		Delete(context.Background(), namespace, v1.DeleteOptions{})

	if err != nil {
		return err
	}

	return nil
}

/*
Deletes PV by its name

Params:

- k8s K8s - an instance of k8s client

- namePV string - name of a PV that is deleated
*/
func deletePV(k8s K8s, namePV string) error {
	err := k8s.clientset.CoreV1().
		PersistentVolumes().
		Delete(context.Background(), namePV, v1.DeleteOptions{})

	if err != nil {
		return err
	}

	return nil

}

/*
Finds inactive namespaces by comparing users enrolled into AWSed and all existing namespaces.

Params:

- controller Controller - an instance of controller

- enrolledUsers []string - a list of usernames of all active AWSed users

- activeNamespaces []string - a list of usernames of all existing namespaces at k8

Returns:

-[]string -   a list of usernames in k8s that are not in AWSed
*/
func diffList(enrolledUsers []string, activeNamespaces []string) []string {

	var diffList []string

	for _, username := range activeNamespaces {
		if !belongsToList(username, enrolledUsers) {
			diffList = append(diffList, username)
		}
	}

	return diffList
}

/*
Checks if a string belongs to a list of strings

Params:

- a string that is being searched for

- a list that is searched at

Returns:

- true if string is a list

- otherwise false
*/
func belongsToList(lookup string, list []string) bool {
	for _, val := range list {

		if val == lookup {
			return true
		}
	}

	return false
}

/*
Cleanes up by deleting inactive namespaces and  PVs

Params:

- controller Controller - an instance of controller
*/
func cleanup(k8s K8s, activeUsers []string, awsed AWSedInterface, dryRun bool) error {

	k8sNames, err := listNamespaces(k8s)

	if err != nil {
		return err
	}

	for _, username := range k8sNames {
		response, err := getUserDetail(username)

		// Error occures only when the request can't be made
		if err != nil {
			return err
		}

		if response >= 300 {
			continue
		} else if belongsToList(username, activeUsers) {
			continue
		}

		log.Println("Will delete namespace", username)

		if !dryRun {
			err := deleteNamespace(k8s, username)

			if err != nil {
				return err
			}
		}
		for _, volumeType := range config.Volumes {

			name := fmt.Sprintf("%v%s", username, volumeType)
			log.Println("Will delete volume", name)
			if !dryRun {
				err := deletePV(k8s, name)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil

}

func main() {

	var k8s K8s
	var awsed AWSed

	activeUsers, err := awsed.getActiveUsers()

	if err != nil {
		log.Fatal(err)
	}

	clientset, err := clientSetup(k8s)

	if err != nil {
		log.Fatal(err)
	}

	k8s.clientset = clientset

	if len(os.Args) > 1 {
		arg := os.Args[1]

		if arg == "--dry-run" {
			err := cleanup(k8s, activeUsers, awsed, true)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Println("Unknown argument")
		}
	} else {
		err := cleanup(k8s, activeUsers, awsed, false)

		if err != nil {
			log.Fatal(err)
		}
	}
}
