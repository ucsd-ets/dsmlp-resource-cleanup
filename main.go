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

type AWSedResponse struct {
	Username    string   `json:"username"`
	FirstName   string   `json:"firstName"`
	LastName    string   `json:"lastName"`
	Uid         int      `json:"uid"`
	Enrollments []string `json:"enrollments"`
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
	getUserEnrollment(name string) (bool, error)
}
type AWSed struct {
}

// We just use usernames, since we don't need Uids or first names
type AWSUser struct {
	Username string `json:"username"`
}

/*
For a given name, makes a get call to https://awsed.ucsd.edu/api/users,
if no awsed record exists or a user is enrolled somewhere, returns false,
otherwise - true

Params:

- name string - a name of user

Returns:

- bool - weather a user is to be deleted

- err
*/
func (a AWSed) getUserEnrollment(name string) (bool, error) {

	var userRecord AWSedResponse
	reqUrl := config.UserUrl + "/" + name
	request, err := http.NewRequest(
		http.MethodGet,
		reqUrl,
		nil,
	)

	if err != nil {
		return false, err
	}

	// Add API key for header
	request.Header.Add("Authorization", awsedApi)

	response, err := http.DefaultClient.Do(request)

	if err != nil {
		return false, err
	}

	code := response.StatusCode

	if code >= 200 {
		return false, nil
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false, fmt.Errorf("error reading HTTP response body: %v", err)
	}

	json.Unmarshal(responseBytes, &userRecord)

	if len(userRecord.Enrollments) > 0 {
		return false, nil
	}

	return true, nil
}

type MockAWSed struct {
	AWSRepo []AWSedResponse
}

/*
Mocks out the enrollment logic with a call to struct var AWSRepo

Params:

* name string - a name of user

Returns:

* bool - weather a user is to be deleted

* err
*/
func (m MockAWSed) getUserEnrollment(name string) (bool, error) {

	for _, response := range m.AWSRepo {
		if response.Username == name {
			if len(response.Enrollments) > 0 {
				return false, nil
			} else {
				return true, nil
			}
		}
	}
	return false, nil
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
Checks if PV is present in k8s clientset

Params:

- k8s K8s - an instance of k8s client

- namePV string - name of a PV that is deleated

Returns:

- boolean value representing weather PV exists or not
*/
func isPvPresent(k8s K8s, namePV string) bool {
	val, err := k8s.clientset.CoreV1().
		PersistentVolumes().
		Get(context.Background(), namePV, v1.GetOptions{})

	if err != nil {
		return false
	}

	return val != nil

}

/*
Cleanes up by deleting inactive namespaces and  PVs

Params:

- controller Controller - an instance of controller
*/
func cleanup(k8s K8s, awsed AWSedInterface, dryRun bool) error {

	k8sNames, err := listNamespaces(k8s)

	if err != nil {
		return err
	}

	for _, username := range k8sNames {
		enrollmentStatus, err := awsed.getUserEnrollment(username)

		// Error occures only when the request can't be made
		if err != nil {
			return err
		}

		if !enrollmentStatus {
			continue
		}

		log.Printf("Will delete namespace: %s \n", username)
		println("Will delete namespace", username)

		if !dryRun {
			err := deleteNamespace(k8s, username)

			if err != nil {
				return err
			}
		}

		for _, volumeType := range config.Volumes {

			name := fmt.Sprintf("%v%s", username, volumeType)
			log.Println("Will delete volume", name)
			println("Will delete volume", name)
			if !dryRun {
				if isPvPresent(k8s, name) {

					err := deletePV(k8s, name)

					if err != nil {
						return err
					}
				} else {
					log.Printf("%s doesn't exist. Skipping \n", name)
					println(name, " doesn't exist. Skipping")
				}
			}
		}

		log.Println("")

	}

	return nil

}

func main() {

	var k8s K8s
	var awsed AWSed

	clientset, err := clientSetup(k8s)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Cleanup started!")
	fmt.Println("Cleanup started")

	k8s.clientset = clientset

	if len(os.Args) > 1 {
		arg := os.Args[1]

		if arg == "--dry-run" {
			err := cleanup(k8s, awsed, true)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Println("Unknown argument")
		}
	} else {
		err := cleanup(k8s, awsed, false)

		if err != nil {
			log.Fatal(err)
		}
	}
}
