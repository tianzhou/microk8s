package util_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/canonical/microk8s/cluster-agent/pkg/util"
	utiltest "github.com/canonical/microk8s/cluster-agent/pkg/util/test"
)

func TestGetServiceArgument(t *testing.T) {
	serviceOneArguments := `
--key=value
--key-with-space value2
   --key-with-padding=value3
--multiple=keys --in-the-same-row=this-is-lost
`
	serviceTwoArguments := `
--key=value-of-service-two
`
	if err := os.MkdirAll("testdata/args", 0755); err != nil {
		t.Fatal("Failed to setup test directory")
	}
	defer os.RemoveAll("testdata/args")
	if err := os.WriteFile("testdata/args/service", []byte(serviceOneArguments), 0644); err != nil {
		t.Fatalf("Failed to create test service arguments: %s", err)
	}
	if err := os.WriteFile("testdata/args/service2", []byte(serviceTwoArguments), 0644); err != nil {
		t.Fatalf("Failed to create test service arguments: %s", err)
	}
	for _, tc := range []struct {
		service       string
		key           string
		expectedValue string
	}{
		{service: "service", key: "--key", expectedValue: "value"},
		{service: "service2", key: "--key", expectedValue: "value-of-service-two"},
		{service: "service", key: "--key-with-padding", expectedValue: "value3"},
		{service: "service", key: "--key-with-space", expectedValue: "value2"},
		{service: "service", key: "--missing", expectedValue: ""},
		{service: "service3", key: "--missing-service", expectedValue: ""},
		// TODO: the final test case documents that arguments in the same row will not be parsed properly.
		// This is carried over from the original Python code, and probably needs fixing in the future.
		{service: "service", key: "--in-the-same-row", expectedValue: ""},
	} {
		t.Run(fmt.Sprintf("%s/%s", tc.service, tc.key), func(t *testing.T) {
			if v := util.GetServiceArgument(tc.service, tc.key); v != tc.expectedValue {
				t.Fatalf("Expected argument value to be %q, but it was %q instead", tc.expectedValue, v)
			}
		})
	}
}

func TestRestart(t *testing.T) {
	m := &utiltest.MockRunner{}
	utiltest.WithMockRunner(m, func(t *testing.T) {
		t.Run("NoKubelite", func(t *testing.T) {
			for _, tc := range []struct {
				service         string
				expectedCommand string
			}{
				{service: "apiserver", expectedCommand: "snapctl restart microk8s.daemon-apiserver"},
				{service: "proxy", expectedCommand: "snapctl restart microk8s.daemon-proxy"},
				{service: "kubelet", expectedCommand: "snapctl restart microk8s.daemon-kubelet"},
				{service: "scheduler", expectedCommand: "snapctl restart microk8s.daemon-scheduler"},
				{service: "controller-manager", expectedCommand: "snapctl restart microk8s.daemon-controller-manager"},
				{service: "kube-apiserver", expectedCommand: "snapctl restart microk8s.daemon-apiserver"},
				{service: "kube-proxy", expectedCommand: "snapctl restart microk8s.daemon-proxy"},
				{service: "kube-scheduler", expectedCommand: "snapctl restart microk8s.daemon-scheduler"},
				{service: "kube-controller-manager", expectedCommand: "snapctl restart microk8s.daemon-controller-manager"},
				{service: "k8s-dqlite", expectedCommand: "snapctl restart microk8s.daemon-k8s-dqlite"},
				{service: "cluster-agent", expectedCommand: "snapctl restart microk8s.daemon-cluster-agent"},
				{service: "containerd", expectedCommand: "snapctl restart microk8s.daemon-containerd"},
			} {
				util.Restart(context.Background(), tc.service)
				if m.CalledWithCommand[len(m.CalledWithCommand)-1] != tc.expectedCommand {
					t.Fatalf("Expected command %q, but %q was called instead", tc.expectedCommand, m.CalledWithCommand)
				}
			}
			for _, service := range []string{"apiserver", "k8s-dqlite"} {
				util.Restart(context.Background(), service)
				expectedCommand := fmt.Sprintf("snapctl restart microk8s.daemon-%s", service)
				if m.CalledWithCommand[len(m.CalledWithCommand)-1] != expectedCommand {
					t.Fatalf("Expected command %q, but %q was called instead", expectedCommand, m.CalledWithCommand)
				}
			}
		})
		t.Run("Kubelite", func(t *testing.T) {
			if err := os.MkdirAll("testdata/var/lock", 0755); err != nil {
				t.Fatalf("Failed to create test directory: %s", err)
			}
			defer os.RemoveAll("testdata/var")
			if _, err := os.Create("testdata/var/lock/lite.lock"); err != nil {
				t.Fatalf("Failed to create kubelite lock file: %s", err)
			}
			for _, tc := range []struct {
				service         string
				expectedCommand string
			}{
				{service: "apiserver", expectedCommand: "snapctl restart microk8s.daemon-kubelite"},
				{service: "proxy", expectedCommand: "snapctl restart microk8s.daemon-kubelite"},
				{service: "kubelet", expectedCommand: "snapctl restart microk8s.daemon-kubelite"},
				{service: "scheduler", expectedCommand: "snapctl restart microk8s.daemon-kubelite"},
				{service: "controller-manager", expectedCommand: "snapctl restart microk8s.daemon-kubelite"},
				{service: "k8s-dqlite", expectedCommand: "snapctl restart microk8s.daemon-k8s-dqlite"},
				{service: "cluster-agent", expectedCommand: "snapctl restart microk8s.daemon-cluster-agent"},
				{service: "containerd", expectedCommand: "snapctl restart microk8s.daemon-containerd"},
			} {
				util.Restart(context.Background(), tc.service)
				if m.CalledWithCommand[len(m.CalledWithCommand)-1] != tc.expectedCommand {
					t.Fatalf("Expected command %q, but %q was called instead", tc.expectedCommand, m.CalledWithCommand)
				}
			}
		})
	})(t)

}

func TestUpdateServiceArguments(t *testing.T) {
	contents := `
--key=value
--other=other-value
--with-space value2
`
	for _, tc := range []struct {
		name           string
		update         map[string]string
		delete         []string
		expectedValues map[string]string
	}{
		{
			name:   "simple-update",
			update: map[string]string{"--key": "new-value"},
			delete: []string{},
			expectedValues: map[string]string{
				"--key":   "new-value",
				"--other": "other-value",
			},
		},
		{
			name:   "update-many-delete-one",
			update: map[string]string{"--key": "new-value", "--other": "other-new-value"},
			delete: []string{"--with-space"},
			expectedValues: map[string]string{
				"--key":        "new-value",
				"--other":      "other-new-value",
				"--with-space": "",
			},
		},
		{
			name: "no-updates",
			expectedValues: map[string]string{
				"--key": "value",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.MkdirAll("testdata/args", 0755); err != nil {
				t.Fatalf("Failed to create test directory: %s", err)
			}
			defer os.RemoveAll("testdata/args")
			if err := os.WriteFile("testdata/args/service", []byte(contents), 0660); err != nil {
				t.Fatalf("Failed to write arguments file: %s", err)
			}

			if err := util.UpdateServiceArguments("service", tc.update, tc.delete); err != nil {
				t.Fatalf("Expected no error updating arguments file but received %q", err)
			}
			for key, expectedValue := range tc.expectedValues {
				if value := util.GetServiceArgument("service", key); value != expectedValue {
					t.Fatalf("Expected value for argument %q does not match (%q and %q)", key, value, expectedValue)
				}
			}
		})
	}
}
