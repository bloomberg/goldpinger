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
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/bloomberg/goldpinger/v3/pkg/models"
)

// checkResults holds the latest results of checking the pods
var checkResults = models.CheckResults{PodResults: make(map[string]models.PodResult)}

// checkResultsMux controls concurrent access to checkResults
var checkResultsMux = sync.Mutex{}

// exists checks whether there is an existing pinger for the given pod
// returns true if:
// - there is already a pinger with the same name
// - the pinger has the same podIP
// - the pinger has the same hostIP
func exists(existingPods map[string]*GoldpingerPod, new *GoldpingerPod) bool {
	old, exists := existingPods[new.Name]
	return exists && (old.PodIP == new.PodIP) && (old.HostIP == new.HostIP)
}

// updatePingers calls SelectPods() at regular intervals to get a new list of goldpinger pods to ping
// For each goldpinger pod, it then creates a pinger responsible for pinging it and returning the
// results on the result channel
func updatePingers(resultsChan chan<- PingAllPodsResult) {
	// Important: This is the only goroutine that should have access to
	// these maps since there is nothing controlling concurrent access
	pingers := make(map[string]*Pinger)
	existingPods := make(map[string]*GoldpingerPod)
	refreshPeriod := time.Duration(GoldpingerConfig.RefreshInterval) * time.Second

	for {
		// Initialize deletedPods to all existing pods, we will remove
		// any pods that should still exist from this list after we are done
		// NOTE: This is *NOT* a copy of existingPods just a new variable name
		// to make the intention/code clear and cleaner
		deletedPods := existingPods

		// New pods are brand new and haven't been seen before
		newPods := make(map[string]*GoldpingerPod)

		latest := SelectPods()
		for podName, pod := range latest {
			if exists(existingPods, pod) {
				// This pod continues to exist in the latest iteration of the update
				// without any changes
				// Delete it from the set of pods that we wish to delete
				delete(deletedPods, podName)
			} else {
				// This pod is brand new and has never been seen before
				// Add it to the list of newPods
				newPods[podName] = pod
			}
		}

		// deletedPods now contains any pods that have either been deleted from the api-server
		// *OR* weren't selected by our rendezvous hash
		// *OR* had their host/pod IP changed. Remove those pingers
		destroyPingers(pingers, deletedPods)

		// Next create pingers for new pods
		createPingers(pingers, newPods, resultsChan, refreshPeriod)

		// Finally, just set existingPods to the latest and collect garbage
		existingPods = latest
		deletedPods = nil
		newPods = nil

		// Wait the given time before pinging
		time.Sleep(refreshPeriod)
	}
}

// createPingers allocates a new pinger object for each new goldpinger Pod that's been discovered
// It also:
//     (a) initializes a result object in checkResults to store info on that pod
//     (b) starts a new goroutines to continuously ping the given pod.
//         Each new goroutine waits for a given time before starting the continuous ping
//         to prevent a thundering herd
func createPingers(pingers map[string]*Pinger, newPods map[string]*GoldpingerPod, resultsChan chan<- PingAllPodsResult, refreshPeriod time.Duration) {
	if len(newPods) == 0 {
		// I have nothing to do
		return
	}
	waitBetweenPods := refreshPeriod / time.Duration(len(newPods))

	zap.L().Info(
		"Starting pingers for new pods",
		zap.Int("numNewPods", len(newPods)),
		zap.Duration("refreshPeriod", refreshPeriod),
		zap.Duration("waitPeriod", waitBetweenPods),
		zap.Float64("JitterFactor", GoldpingerConfig.JitterFactor),
	)

	initialWait := time.Duration(0)
	for podName, pod := range newPods {
		pinger := NewPinger(pod, resultsChan)
		pingers[podName] = pinger
		go pinger.PingContinuously(initialWait, refreshPeriod, GoldpingerConfig.JitterFactor)
		initialWait += waitBetweenPods
	}
}

// destroyPingers takes a list of deleted pods and then for each pod in the list, it stops
// the goroutines that continuously pings that pod and then deletes the pod from the list of pingers
func destroyPingers(pingers map[string]*Pinger, deletedPods map[string]*GoldpingerPod) {
	for podName, pod := range deletedPods {
		zap.L().Info(
			"Deleting pod from pingers",
			zap.String("name", podName),
			zap.String("podIP", pod.PodIP),
			zap.String("hostIP", pod.HostIP),
		)
		pinger := pingers[podName]

		// Close the channel to stop pinging
		close(pinger.stopChan)

		// delete from pingers
		delete(pingers, podName)
	}
}

// updateCounters updates the count of health and unhealthy nodes
func updateCounters() {
	checkResultsMux.Lock()
	defer checkResultsMux.Unlock()

	var counterHealthy float64
	for _, result := range checkResults.PodResults {
		if result.OK != nil && *result.OK {
			counterHealthy++
		}
	}
	CountHealthyUnhealthyNodes(counterHealthy, float64(len(checkResults.PodResults))-counterHealthy)
}

// collectResults simply reads results from the results channel and saves them in a map
func collectResults(resultsChan <-chan PingAllPodsResult) {
	refreshPeriod := time.Duration(GoldpingerConfig.RefreshInterval) * time.Second
	updateTicker := time.NewTicker(refreshPeriod)
	for {
		select {
		case <-updateTicker.C:
			// Every time our update ticker ticks, update the count of healthy/unhealthy nodes
			updateCounters()
		case response := <-resultsChan:
			// On getting a ping response, if the pinger is not being deleted,
			// simply save it for later
			checkResultsMux.Lock()
			if response.deleted {
				delete(checkResults.PodResults, response.podName)
			} else {
				checkResults.PodResults[response.podName] = response.podResult
			}
			checkResultsMux.Unlock()
		}
	}
}

func StartUpdater() {
	if GoldpingerConfig.RefreshInterval <= 0 {
		zap.L().Info("Not creating updater, refresh interval is negative", zap.Int("RefreshInterval", GoldpingerConfig.RefreshInterval))
		return
	}

	pods := SelectPods()

	// Create a channel for the results
	resultsChan := make(chan PingAllPodsResult, len(pods))
	go updatePingers(resultsChan)
	go collectResults(resultsChan)
}
