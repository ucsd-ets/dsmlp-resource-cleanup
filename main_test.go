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

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

var mockK8s K8s
var mockAWS MockAWSed

func setupTest() {
	log.Println("setup suite")
	mockK8s.clientset = testclient.NewSimpleClientset()
	clientset := mockK8s.clientset
	mockAwsRepo, err := getAwsJson()
	mockAWS = MockAWSed{AWSRepo: mockAwsRepo}

	if err != nil {
		log.Fatal(err)
	}

	// creats users in mock k8s clientset that are in mock_AWS.json
	for _, user := range mockAwsRepo {
		namespace := &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: user.Username,
			},
		}

		clientset.CoreV1().Namespaces().Create(context.Background(), namespace, v1.CreateOptions{})
		// creates volumes according to the volume type
		pv := &corev1.PersistentVolume{
			ObjectMeta: v1.ObjectMeta{
				Name: user.Username,
			}}

		clientset.CoreV1().PersistentVolumes().Create(context.Background(), pv, v1.CreateOptions{})

	}

	// creates a namespaces and associated PVs that are not in AWSed roster
	namespace := &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: "dvader",
		},
	}
	clientset.CoreV1().Namespaces().Create(context.Background(), namespace, v1.CreateOptions{})

	pv := &corev1.PersistentVolume{
		ObjectMeta: v1.ObjectMeta{
			Name: "dvader",
		}}

	clientset.CoreV1().PersistentVolumes().Create(context.Background(), pv, v1.CreateOptions{})
}

func TestMain(m *testing.M) {
	setupTest()
	code := m.Run()
	// shutdown()
	os.Exit(code)
}

/*Testing out enrollment logic*/
func TestGetUserEnrollmentNotInRepo(t *testing.T) {
	got, err := mockAWS.getUserEnrollment("dvadre")

	if err != nil {
		log.Fatal(err)
	}
	expected := false

	if got != expected {
		t.Errorf("User not in repo to delete")

		fmt.Println("Expected", expected)
		fmt.Println("Got", got)
	}
}

func TestGetUserEnrollmentInRepoShouldFalse(t *testing.T) {
	got, err := mockAWS.getUserEnrollment("tix034")

	if err != nil {
		log.Fatal(err)
	}

	expected := false

	if got != expected {
		t.Errorf("User in repo to delete")

		fmt.Println("Expected", expected)
		fmt.Println("Got", got)
	}
}

func TestGetUserEnrollmentInRepoShouldTrue(t *testing.T) {
	got, err := mockAWS.getUserEnrollment("pbotros")

	if err != nil {
		log.Fatal(err)
	}

	expected := true

	if got != expected {
		t.Errorf("User in repo to keep")

		fmt.Println("Expected", expected)
		fmt.Println("Got", got)
	}
}

/* Test namespace logic */
func TestListNamespaces(t *testing.T) {
	log.Println("TestListNamespaces running")

	expected := []string{"btice", "dvader", "pbotros", "n2nazar", "tix034"}
	got, err := listNamespaces(mockK8s)

	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(expected)
	sort.Strings(got)

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("lists don't match")

		fmt.Println("Expected", expected)
		fmt.Println("Got", got)
	}

	log.Println("TestListNamespaces Ok")
}

func TestDeleteNamespace(t *testing.T) {

	log.Println("TestDeleteNamespace running")

	expected := []string{"btice", "pbotros", "n2nazar", "tix034"}
	deleteNamespace(mockK8s, "dvader")

	got, err := listNamespaces(mockK8s)

	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(expected)
	sort.Strings(got)

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

/* Test JSON reader */
func TestGetAwsJson(t *testing.T) {
	log.Println()
	log.Println("Test Get Aws JSON mock function")

	awsRepo, err := getAwsJson()

	if err != nil {
		return
	}

	var expectedEnrollments []string
	expectedEnrollments = append(expectedEnrollments, "MUS206_WI23_D00")

	expected := AWSedResponse{Username: "btice", FirstName: "brian", LastName: "tice", Uid: 130507, Enrollments: expectedEnrollments}

	got := awsRepo[0]

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("lists don't match")

		fmt.Println("Expected", expected)
		fmt.Println("Got", got)
	}

	log.Print("Ok")
	log.Println()
}

func getAwsJson() ([]AWSedResponse, error) {
	var awsRepo []AWSedResponse

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
