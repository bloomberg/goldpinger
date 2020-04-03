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
	"time"

	"go.uber.org/zap"

	"github.com/bloomberg/goldpinger/v3/pkg/models"
)

// checkResults holds the latest results of checking the pods
var checkResults models.CheckResults

// counterHealthy is the number of healthy pods
var counterHealthy float64

// getPingers creates a new set of pingers for the given pods
// Each pinger is responsible for pinging a single pod and returns
// the results on the results channel
func getPingers(pods map[string]*GoldpingerPod, resultsChan chan<- PingAllPodsResult) map[string]*Pinger {
	pingers := map[string]*Pinger{}

	for podName, pod := range pods {
		pingers[podName] = NewPinger(pod, resultsChan)
	}
	return pingers
}

// initCheckResults initializes the check results, which will be updated continuously
// as the results come in
func initCheckResults(pingers map[string]*Pinger) {
	checkResults = models.CheckResults{}
	checkResults.PodResults = make(map[string]models.PodResult)
	for podName, pinger := range pingers {
		checkResults.PodResults[podName] = pinger.result.podResult
	}
	counterHealthy = 0
}

// startPingers starts `n` goroutines to continuously ping all the given pods, one goroutine per pod
// It staggers the start of all the go-routines to prevent a thundering herd
func startPingers(pingers map[string]*Pinger) {
	refreshPeriod := time.Duration(GoldpingerConfig.RefreshInterval) * time.Second
	waitBetweenPods := refreshPeriod / time.Duration(len(pingers))

	zap.L().Info(
		"Starting Pingers",
		zap.Duration("refreshPeriod", refreshPeriod),
		zap.Duration("waitPeriod", waitBetweenPods),
		zap.Float64("JitterFactor", GoldpingerConfig.JitterFactor),
	)

	for _, p := range pingers {
		go p.PingContinuously(refreshPeriod, GoldpingerConfig.JitterFactor)
		time.Sleep(waitBetweenPods)
	}
}

// updateCounters updates the value of health and unhealthy nodes as the results come in
func updateCounters(podName string, result *models.PodResult) {
	// Get the previous value of ok
	old := checkResults.PodResults[podName]
	oldOk := (old.OK != nil && *old.OK)

	// Check if the value of ok has changed
	// If not, do nothing
	if oldOk == *result.OK {
		return
	}

	if *result.OK {
		// The value was previously false and just became true
		// Increment the counter
		counterHealthy++
	} else {
		// The value was previously true and just became false
		counterHealthy--
	}
	CountHealthyUnhealthyNodes(counterHealthy, float64(len(checkResults.PodResults))-counterHealthy)
}

// collectResults simply reads results from the results channel and saves them in a map
func collectResults(resultsChan <-chan PingAllPodsResult) {
	go func() {
		for response := range resultsChan {
			result := response.podResult
			updateCounters(response.podName, &result)
			checkResults.PodResults[response.podName] = result
		}
	}()
}

func StartUpdater() {
	if GoldpingerConfig.RefreshInterval <= 0 {
		zap.L().Info("Not creating updater, refresh interval is negative", zap.Int("RefreshInterval", GoldpingerConfig.RefreshInterval))
		return
	}

	pods := SelectPods()
	zap.S().Infof("Got Pods: %+v", pods)

	// Create a channel for the results
	resultsChan := make(chan PingAllPodsResult, len(pods))
	pingers := getPingers(pods, resultsChan)
	initCheckResults(pingers)
	startPingers(pingers)
	collectResults(resultsChan)
}
