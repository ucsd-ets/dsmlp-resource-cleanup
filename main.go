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

// TODO: check that it's not null in main
var config, _ = loadConfig("config.json")

var volumes = []string{
	"-dsmlp-datasets",
	"-dsmlp-datasets-nfs",
	"-home",
	"-home-nfs",
	"-nbgrader",
	"-support",
	"-teams",
}

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
	getEnrollments() ([]string, error)
}
type AWSed struct {
}

// We just use usernames, since we don't need Uids or first names
type ActiveUser struct {
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
func (a AWSed) getEnrollments() ([]string, error) {
	var activeUsersNames []string
	var activeUsers []ActiveUser
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
		return activeUsersNames, err
	}

	// Add API key for header
	request.Header.Add("Authorization", awsedApi)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return activeUsersNames, fmt.Errorf("error reading HTTP response body: %v", err)
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return activeUsersNames, fmt.Errorf("error reading HTTP response body: %v", err)
	}

	json.Unmarshal(responseBytes, &activeUsers)

	for _, user := range activeUsers {
		activeUsersNames = append(activeUsersNames, user.Username)
	}

	return activeUsersNames, err

}

type MockAWSed struct {
}

func (m MockAWSed) getEnrollments() ([]string, error) {
	var activeUsersNames []string
	var activeUsers []ActiveUser
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
	listNamespace()
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
func clientSetup(k8s K8s) {

	config, err := rest.InClusterConfig()

	if err != nil {
		fmt.Println(err)
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		fmt.Println(err)
	}

	k8s.clientset = clientset
}

/*
Creates a list of names of actives namespaces

Params:

- k8s K8s - an instance of k8s client

Returns:

- []string - a list of all active namespaces in cluster
*/
func listNamespace(k8s K8s) []string {

	var dslmpNamespacelist []string

	namspaceList, err := k8s.clientset.CoreV1().
		Namespaces().
		List(context.Background(), v1.ListOptions{})

	if err != nil {
		fmt.Println(err)
	}

	for _, n := range namspaceList.Items {
		dslmpNamespacelist = append(dslmpNamespacelist, n.Name)
	}

	return dslmpNamespacelist
}

/*
Deletes a namespace by name

Params:

- k8s K8s - an instance of k8s client

- namespace string - name of a namespace that is deleated
*/
func deleateNamespace(k8s K8s, namespace string) {
	err := k8s.clientset.CoreV1().
		Namespaces().
		Delete(context.Background(), namespace, v1.DeleteOptions{})

	if err != nil {
		fmt.Println(err)
	}
}

/*
Deletes PV by its name

Params:

- k8s K8s - an instance of k8s client

- namePV string - name of a PV that is deleated
*/
func deletePV(k8s K8s, namePV string) {
	err := k8s.clientset.CoreV1().
		PersistentVolumes().
		Delete(context.Background(), namePV, v1.DeleteOptions{})

	if err != nil {
		fmt.Println(err)
	}

}

type MockK8s struct {
}

type Controller struct {
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
func diffList(controller Controller, enrolledUsers []string, activeNamespaces []string) []string {

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
func cleanup(controller Controller, k8s K8s, awsed AWSedInterface) {

	enrolledUsers, err := awsed.getEnrollments()

	if err != nil {
		log.Fatal(err)
	}
	activeNamespaces := listNamespace(k8s)

	inactiveNames := diffList(controller, enrolledUsers, activeNamespaces)
	fmt.Println(inactiveNames)

	for _, username := range inactiveNames {
		fmt.Printf("Will delete namespace %v", username)
		deleateNamespace(k8s, username)

		for _, volumeType := range volumes {
			name := fmt.Sprintf("%v%s", username, volumeType)
			fmt.Printf("Will delete volume %v", name)
			deletePV(k8s, name)
		}
	}
}

/*
Lists all namespaces and  PVs that would be cleaned up

Params:

- controller Controller - an instance of controller
*/
func drycleanup(controller Controller) {
	var awsed AWSed
	var k8s K8s

	clientSetup(k8s)

	enrolledUsers, err := awsed.getEnrollments()

	if err != nil {
		log.Fatal(err)
	}
	activeNamespaces := listNamespace(k8s)

	inactiveNames := diffList(controller, enrolledUsers, activeNamespaces)

	for _, username := range inactiveNames {
		notify := fmt.Sprintf("Will delete namespace %v", username)
		fmt.Println("Will delete volume", notify)

		for _, volumeType := range volumes {
			name := fmt.Sprintf("%v%s", username, volumeType)
			fmt.Println("Will delete volume", name)
		}
	}
}

func main() {
	var controller Controller
	var awsed AWSed

	var k8s K8s
	clientSetup(k8s)

	if len(os.Args) > 0 {
		arg := os.Args[0]

		if arg == "-dry-run" {
			drycleanup(controller)
		} else {
			fmt.Println("Unknown argument")
		}
	} else {
		cleanup(controller, k8s, awsed)
	}
}
