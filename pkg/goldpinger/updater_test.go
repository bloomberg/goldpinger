// Copyright 2018 Bloomberg Finance L.P.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goldpinger

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	testNamespace     = "goldpinger"
	testLabelSelector = "app=goldpinger"
	// testRefreshPeriod is long enough that pingers started by a test ping
	// at most once before the test is over.
	testRefreshPeriod = time.Hour
)

// goldpingerPodObject returns a running goldpinger pod as the API server
// would list it. Its IPs are in TEST-NET-1, so pings to it go nowhere.
func goldpingerPodObject(name, podIP, hostIP string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			Labels:    map[string]string{"app": "goldpinger"},
		},
		Status: v1.PodStatus{
			Phase:  v1.PodRunning,
			PodIP:  podIP,
			HostIP: hostIP,
		},
	}
}

// useFakeAPIServer points GoldpingerConfig at a fake API server serving the
// given pods. Listing pods fails while the returned flag is set.
func useFakeAPIServer(t *testing.T, pods ...runtime.Object) (*fake.Clientset, *atomic.Bool) {
	t.Helper()

	client := fake.NewSimpleClientset(pods...)
	listFails := &atomic.Bool{}
	client.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		if listFails.Load() {
			return true, nil, errors.New("http2: client connection lost")
		}
		return false, nil, nil
	})

	originalConfig := GoldpingerConfig
	t.Cleanup(func() { GoldpingerConfig = originalConfig })
	namespace := testNamespace
	GoldpingerConfig.KubernetesClient = client
	GoldpingerConfig.Namespace = &namespace
	GoldpingerConfig.LabelSelector = testLabelSelector
	GoldpingerConfig.IPVersions = []string{"4"}
	GoldpingerConfig.Hostname = testHostname
	GoldpingerConfig.PingNumber = 0
	GoldpingerConfig.PingTimeout = 10 * time.Millisecond

	return client, listFails
}

// startResultsDrain consumes everything pingers send until the test is over.
// It returns the results channel to give to pingers, and a channel that gets
// the name of every pinger that has stopped.
func startResultsDrain(t *testing.T) (chan<- PingAllPodsResult, <-chan string) {
	t.Helper()
	resultsChan := make(chan PingAllPodsResult)
	stopped := make(chan string, 16)
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })
	go func() {
		for {
			select {
			case result := <-resultsChan:
				if result.deleted {
					stopped <- result.podName
				}
			case <-done:
				return
			}
		}
	}()
	return resultsChan, stopped
}

// stopPingers stops the remaining pingers and waits until the goroutines of
// all pingers named in started are done, so they don't read GoldpingerConfig
// while a test restores it.
func stopPingers(t *testing.T, pingers map[string]*Pinger, started []string, stopped <-chan string) {
	t.Helper()
	destroyPingers(pingers, syncedPods(pingers))
	waiting := make(map[string]bool, len(started))
	for _, name := range started {
		waiting[name] = true
	}
	for len(waiting) > 0 {
		select {
		case name := <-stopped:
			delete(waiting, name)
		case <-time.After(5 * time.Second):
			t.Fatalf("pingers did not stop: %v", waiting)
		}
	}
}

func TestGetAllPodsReturnsListError(t *testing.T) {
	_, listFails := useFakeAPIServer(t, goldpingerPodObject("goldpinger-a", "192.0.2.1", "198.51.100.1"))

	pods, err := GetAllPods()
	if err != nil {
		t.Fatalf("GetAllPods returned error %v, want none", err)
	}
	if len(pods) != 1 {
		t.Fatalf("GetAllPods returned %d pods, want 1", len(pods))
	}

	listFails.Store(true)
	pods, err = GetAllPods()
	if err == nil {
		t.Fatalf("GetAllPods returned no error while listing pods fails")
	}
	if pods != nil {
		t.Errorf("GetAllPods returned %d pods with an error, want nil", len(pods))
	}
}

// TestSyncPingersKeepsPingersWhenListFails verifies that a failing pod list
// doesn't look like all peers are gone: the pingers (and so their metrics)
// must survive until the pods can be listed again.
func TestSyncPingersKeepsPingersWhenListFails(t *testing.T) {
	client, listFails := useFakeAPIServer(t,
		goldpingerPodObject("goldpinger-a", "192.0.2.1", "198.51.100.1"),
		goldpingerPodObject("goldpinger-b", "192.0.2.2", "198.51.100.2"),
	)
	resultsChan, stopped := startResultsDrain(t)
	pingers := make(map[string]*Pinger)
	t.Cleanup(func() { stopPingers(t, pingers, []string{"goldpinger-a", "goldpinger-b"}, stopped) })

	existing := syncPingers(pingers, map[string]*GoldpingerPod{}, resultsChan, testRefreshPeriod)
	if len(pingers) != 2 || len(existing) != 2 {
		t.Fatalf("after first sync: %d pingers, %d pods, want 2 and 2", len(pingers), len(existing))
	}
	stopChanA := pingers["goldpinger-a"].stopChan

	listFails.Store(true)
	existing = syncPingers(pingers, existing, resultsChan, testRefreshPeriod)
	if len(pingers) != 2 || len(existing) != 2 {
		t.Fatalf("after failed list: %d pingers, %d pods, want 2 and 2", len(pingers), len(existing))
	}
	select {
	case <-stopChanA:
		t.Fatalf("pinger for goldpinger-a was stopped after a failed list")
	default:
	}

	// Once listing works again, pods that are really gone are still removed
	listFails.Store(false)
	if err := client.CoreV1().Pods(testNamespace).Delete(t.Context(), "goldpinger-b", metav1.DeleteOptions{}); err != nil {
		t.Fatalf("could not delete pod from fake API server: %v", err)
	}
	existing = syncPingers(pingers, existing, resultsChan, testRefreshPeriod)
	if len(pingers) != 1 || len(existing) != 1 {
		t.Fatalf("after pod deletion: %d pingers, %d pods, want 1 and 1", len(pingers), len(existing))
	}
	if _, ok := pingers["goldpinger-a"]; !ok {
		t.Errorf("pinger for goldpinger-a is gone, want it kept")
	}
}

// syncedPods returns the pods the given pingers are pinging.
func syncedPods(pingers map[string]*Pinger) map[string]*GoldpingerPod {
	pods := make(map[string]*GoldpingerPod, len(pingers))
	for name, p := range pingers {
		pods[name] = p.pod
	}
	return pods
}
