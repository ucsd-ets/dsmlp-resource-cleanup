package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"sort"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var mockK8s K8s
var mockAWS MockAWSed

// TODO: do setup
func setupTest() {
	//mockK8s.clientset = testclient.NewSimpleClientset()
	log.Println("setup suite")

}

func TestMain(m *testing.M) {
	setupTest()
	code := m.Run()
	// shutdown()
	os.Exit(code)
}

// TODO: do enrollments
func getUserEnrollment(t *testing.T) {

}

func TestListNamespaces(t *testing.T) {
	log.Println("TestListNamespaces running")

	expected := []string{"asavarapu", "dvader", "aanil", "rhecuba", "n2nazar", "mkay"}
	got, err := listNamespaces(mockK8s)

	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(expected)

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("lists don't match")

		fmt.Println("Expected", expected)
		fmt.Println("Got", got)
	}

	log.Println("TestListNamespaces Ok")

}

func TestDeleteNamespace(t *testing.T) {

	log.Println("TestDeleteNamespace running")

	expected := []string{"asavarapu", "aanil", "rhecuba", "n2nazar", "mkay"}
	deleteNamespace(mockK8s, "dvader")

	got, err := listNamespaces(mockK8s)

	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(expected)

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("lists don't match")

		fmt.Println("Expected", expected)
		fmt.Println("Got", got)
	}

	log.Println("TestDeleteNamespace Ok")
}

func TestDeletePV(t *testing.T) {

	log.Println("TestDeletePV running")

	expected := []string{"asavarapu", "aanil", "rhecuba", "n2nazar", "mkay"}

	deletePV(mockK8s, "dvader")

	var got []string

	namspaceList, err := mockK8s.clientset.CoreV1().
		PersistentVolumes().
		List(context.Background(), v1.ListOptions{})

	if err != nil {
		t.Errorf("lists don't match")
	}

	for _, n := range namspaceList.Items {
		got = append(got, n.Name)
	}

	sort.Strings(expected)
	sort.Strings(got)

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("lists don't match")

		fmt.Println("Expected", expected)
		fmt.Println("Got", got)
	}

	log.Println("TestDeletePV Ok")
}

func TestGetAwsJson(t *testing.T) {
	log.Println()
	log.Println("Test Get Aws JSON mock function")

	awsRepo, err := getAwsJson()

	if err != nil {
		return
	}

	var expectedEnrollments []string
	expectedEnrollments = append(expectedEnrollments, "MUS206_WI23_D00")

	expected := AWSRecord{Username: "btice", FirstName: "brian", LastName: "tice", Uid: 130507, Enrollments: expectedEnrollments}

	got := awsRepo[0]

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("lists don't match")

		fmt.Println("Expected", expected)
		fmt.Println("Got", got)
	}

	log.Print("Ok")
	log.Println()
}

type AWSRecord struct {
	Username    string   `json:"username"`
	FirstName   string   `json:"firstName"`
	LastName    string   `json:"lastName"`
	Uid         int      `json:"uid"`
	Enrollments []string `json:"enrollments"`
}

func getAwsJson() ([]AWSRecord, error) {
	var awsRepo []AWSRecord

	userFile, err := os.Open("tests/mock_AWS.json")

	if err != nil {
		return nil, err
	}

	defer userFile.Close()

	responseBytes, err := ioutil.ReadAll(userFile)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(responseBytes, &awsRepo)

	return awsRepo, nil
}
