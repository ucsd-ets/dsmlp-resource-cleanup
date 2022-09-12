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
		for _, volumeType := range config.Volumes {
			name := fmt.Sprintf("%v%s", username, volumeType)

			pv := &corev1.PersistentVolume{
				ObjectMeta: v1.ObjectMeta{
					Name: name,
				}}

			clientset.CoreV1().PersistentVolumes().Create(context.Background(), pv, v1.CreateOptions{})
		}

	}

	// creates a namespaces and associated PVs that are not in AWSed roster
	namespace := &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: "dvader",
		},
	}
	clientset.CoreV1().Namespaces().Create(context.Background(), namespace, v1.CreateOptions{})

	for _, volumeType := range config.Volumes {
		name := fmt.Sprintf("%v%s", "dvader", volumeType)

		pv := &corev1.PersistentVolume{
			ObjectMeta: v1.ObjectMeta{
				Name: name,
			}}

		clientset.CoreV1().PersistentVolumes().Create(context.Background(), pv, v1.CreateOptions{})
	}

}

func TestMain(m *testing.M) {
	setupTest()
	code := m.Run()
	// shutdown()
	os.Exit(code)
}

func TestDiffList(t *testing.T) {
	namespaces, err := listNamespace(mockK8s)

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

func TestCleanup(t *testing.T) {

	cleanup(mockK8s, mockAWS, true)

	namespaces, err := listNamespace(mockK8s)

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
