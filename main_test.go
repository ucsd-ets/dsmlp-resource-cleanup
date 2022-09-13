package main

import (
	"context"
	"fmt"
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
	mockK8s.clientset = testclient.NewSimpleClientset()
	log.Println("setup suite")

	clientset := mockK8s.clientset

	usernames, err := mockAWS.getEnrollments()

	if err != nil {
		log.Fatal(err)
	}

	// loop that creates namespaces and PVs from mock_AWS.json
	for _, username := range usernames {
		namespace := &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: username,
			},
		}

		clientset.CoreV1().Namespaces().Create(context.Background(), namespace, v1.CreateOptions{})
		// creates volumes according to the volume type
		pv := &corev1.PersistentVolume{
			ObjectMeta: v1.ObjectMeta{
				Name: username,
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

func TestDiffList(t *testing.T) {
	namespaces, err := listNamespaces(mockK8s)

	if err != nil {
		log.Fatal(err)
	}

	enrolledStd, err := mockAWS.getEnrollments()

	if err != nil {
		log.Fatal(err)
	}

	diffList := diffList(enrolledStd, namespaces)

	expected := []string{"dvader"}

	if !reflect.DeepEqual(diffList, expected) {
		t.Errorf("lists don't match")

		fmt.Println(diffList)
		fmt.Println(expected)
	}
}

func TestListNamespaces(t *testing.T) {
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

}

func TestDeleteNamespace(t *testing.T) {
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
}

func TestDeletePV(t *testing.T) {

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
}

func TestCleanup(t *testing.T) {

	cleanup(mockK8s, mockAWS, false)

	namespaces, err := listNamespaces(mockK8s)

	if err != nil {
		log.Fatal(err)
	}

	enrolledStd, err := mockAWS.getEnrollments()

	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(enrolledStd)

	if !reflect.DeepEqual(namespaces, enrolledStd) {
		t.Errorf("lists don't match")

		fmt.Println(namespaces)
		fmt.Println(enrolledStd)
	}

}

// ! Test doesn't work
func TestGetEnrollments(t *testing.T) {
	var awsed AWSed

	enrolledStd, err := awsed.getEnrollments()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(awsed.getEnrollments())
	fmt.Println(enrolledStd)

	if len(enrolledStd) == 0 {
		t.Errorf("Student list is empty. Possible API faliure")
	}
}
