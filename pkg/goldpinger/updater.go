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
	"context"
	"fmt"
	"time"
	"sync"

	"go.uber.org/zap"

	"github.com/bloomberg/goldpinger/pkg/models"
	"github.com/go-openapi/strfmt"
)

// resultsMux controls access to the results from multiple goroutines
var resultsMux sync.Mutex

// getPingers creates a new set of pingers for the given pods
// Each pinger is responsible for pinging a single pod and returns
// the results on the results channel
func getPingers(pods map[string]string, resultsChan chan<- PingAllPodsResult) map[string]*Pinger {
	pingers := map[string]*Pinger{}

	for podIP, hostIP := range pods {
		pingers[podIP] = NewPinger(podIP, hostIP, resultsChan)
	}
	return pingers
}

// startPingers starts `n` goroutines to continuously ping all the given pods, one goroutine per pod
// It staggers the start of all the go-routines to prevent a thundering herd
func startPingers(pingers map[string]*Pinger) {
	refreshPeriod := time.Duration(GoldpingerConfig.RefreshInterval) * time.Second
	waitBetweenPods := refreshPeriod / time.Duration(len(pingers))

	log.Printf("Refresh Period: %+v Wait Period: %+v Jitter Factor: %+v", refreshPeriod, waitBetweenPods, GoldpingerConfig.JitterFactor)

	for _, p := range pingers {
		go p.PingContinuously(refreshPeriod, GoldpingerConfig.JitterFactor)
		time.Sleep(waitBetweenPods)
	}
}

// collectResults simply reads results from the results channel and saves them in a map
func collectResults(resultsChan <-chan PingAllPodsResult) *models.CheckResults {
	results := models.CheckResults{}
	results.PodResults = make(map[string]models.PodResult)
	go func() {
		for response := range resultsChan {
			var podIPv4 strfmt.IPv4
			podIPv4.UnmarshalText([]byte(response.podIP))

			// results.PodResults map will be read by processResults()
			// to count the number of healthy and unhealthy nodes
			// Since concurrent access to maps isn't safe from multiple
			// goroutines, lock the mutex before update
			resultsMux.Lock()
			results.PodResults[response.podIP] = response.podResult
			resultsMux.Unlock()
		}
	}()
	return &results
}

// processResults goes through all the entries in the results channel and counts
// the number of health and unhealth nodes. It just reports the correct number
func processResults(results *models.CheckResults) {
	for {
		var troublemakers []string
		var counterHealthy, counterUnhealthy float64

		resultsMux.Lock()
		for podIP, value := range results.PodResults {
			if *value.OK != true {
				counterUnhealthy++
				troublemakers = append(troublemakers, fmt.Sprintf("%s (%s)", podIP, value.HostIP.String()))
			} else {
				counterHealthy++
			}
		}
		resultsMux.Unlock()

		CountHealthyUnhealthyNodes(counterHealthy, counterUnhealthy)
		if len(troublemakers) > 0 {
			log.Println("Updater ran into trouble with these peers: ", troublemakers)
		}
		time.Sleep(time.Duration(GoldpingerConfig.RefreshInterval) * time.Second)
	}
}

func StartUpdater() {
	if GoldpingerConfig.RefreshInterval <= 0 {
		zap.L().Info("Not creating updater, refresh interval is negative", zap.Int("RefreshInterval", GoldpingerConfig.RefreshInterval))
		return
	}

	pods := GoldpingerConfig.PodSelecter.SelectPods()
	zap.S().Infof("Got Pods: %+v", pods)

	// Create a channel for the results
	resultsChan := make(chan PingAllPodsResult, len(pods))
	pingers := getPingers(pods, resultsChan)

	startPingers(pingers)
	results := collectResults(resultsChan)
	go processResults(results)
}
