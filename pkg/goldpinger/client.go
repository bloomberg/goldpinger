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
	"errors"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	apiclient "github.com/bloomberg/goldpinger/v3/pkg/client"
	"github.com/bloomberg/goldpinger/v3/pkg/client/operations"
	"github.com/bloomberg/goldpinger/v3/pkg/models"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"go.uber.org/zap"
)

// CheckNeighbours queries the kubernetes API server for all other goldpinger pods
// then calls Ping() on each one
func CheckNeighbours(ctx context.Context) *models.CheckResults {
	// Mux to prevent concurrent map address
	checkResultsMux.Lock()
	defer checkResultsMux.Unlock()
	final := models.CheckResults{}
	final.PodResults = make(map[string]models.PodResult)
	for podName, podResult := range checkResults.PodResults {
		final.PodResults[podName] = podResult
	}
	final.ProbeResults = checkTargets()
	return &final
}

// CheckNeighboursNeighbours queries the kubernetes API server for all other goldpinger
// pods then calls Check() on each one
func CheckNeighboursNeighbours(ctx context.Context) *models.CheckAllResults {
	return CheckAllPods(ctx, SelectPods())
}

// CheckCluster does a CheckNeighboursNeighbours and analyses results to produce a binary OK or not OK
func CheckCluster(ctx context.Context) *models.ClusterHealthResults {
	start := time.Now()
	output := models.ClusterHealthResults{
		GeneratedAt: strfmt.DateTime(start),
		OK:          true,
	}
	selectedPods := SelectPods()

	// precompute the expected set of nodes
	expectedNodes := []string{}
	for _, peer := range selectedPods {
		expectedNodes = append(expectedNodes, peer.HostIP)
	}
	sort.Strings(expectedNodes)

	// get the response we serve for check_all
	checkAll := CheckAllPods(ctx, selectedPods)

	// we should at the very least have a response from ourselves
	if len(checkAll.Responses) < 1 {
		output.OK = false
	}
	for _, resp := range checkAll.Responses {
		// 1. check that all nodes report OK
		if *resp.OK {
			output.NodesHealthy = append(output.NodesHealthy, resp.HostIP.String())
		} else {
			output.NodesUnhealthy = append(output.NodesUnhealthy, resp.HostIP.String())
			output.OK = false
		}
		output.NodesTotal++
		// 2. check that all nodes report the expected peers
		// on error, there might be no response from the node
		if resp.Response == nil {
			output.OK = false
			continue
		}
		// if we get a response, let's check we get the expected nodes
		observedNodes := []string{}
		for _, peer := range resp.Response.PodResults {
			observedNodes = append(observedNodes, string(peer.HostIP))
		}
		sort.Strings(observedNodes)
		if len(observedNodes) != len(expectedNodes) {
			output.OK = false
		}
		for i, val := range observedNodes {
			if val != expectedNodes[i] {
				output.OK = false
				break
			}
		}
	}
	output.DurationNs = time.Since(start).Nanoseconds()
	return &output
}

// PingAllPodsResult holds results from pinging all nodes
type PingAllPodsResult struct {
	podName   string
	podResult models.PodResult
	deleted   bool
}

func pickPodHostIP(podIP, hostIP string) string {
	if GoldpingerConfig.UseHostIP {
		return hostIP
	}
	return podIP
}

func checkTargets() models.ProbeResults {
	results := make(map[string][]models.ProbeResult)
	probes := []struct {
		protocol string
		hosts    []string
		probeFn  func(addr string, timeout time.Duration) error
		statFn   func(host string)
		timeout  time.Duration
	}{
		{
			protocol: "dns",
			hosts:    GoldpingerConfig.DnsHosts,
			probeFn:  doDNSProbe,
			statFn:   CountDnsError,
			timeout:  GoldpingerConfig.DnsCheckTimeout,
		},
		{
			protocol: "http",
			hosts:    GoldpingerConfig.HTTPTargets,
			probeFn:  doHTTPProbe,
			statFn:   CountHttpError,
			timeout:  GoldpingerConfig.HTTPCheckTimeout,
		},
		{
			protocol: "tcp",
			hosts:    GoldpingerConfig.TCPTargets,
			probeFn:  doTCPProbe,
			statFn:   CountTcpError,
			timeout:  GoldpingerConfig.TCPCheckTimeout,
		},
	}

	for _, probe := range probes {
		for _, host := range probe.hosts {
			if _, ok := results[host]; !ok {
				results[host] = []models.ProbeResult{}
			}

			res := models.ProbeResult{Protocol: probe.protocol}
			start := time.Now()
			err := probe.probeFn(host, probe.timeout)
			if err != nil {
				res.Error = err.Error()
				probe.statFn(host)
			}

			res.ResponseTimeMs = time.Since(start).Milliseconds()
			results[host] = append(results[host], res)
		}
	}

	return results
}

// CheckServicePodsResult results of the /check operation
type CheckServicePodsResult struct {
	podName           string
	checkAllPodResult models.CheckAllPodResult
	hostIPv4          strfmt.IPv4
	podIPv4           strfmt.IPv4
}

// CheckAllPods calls all neighbours and returns a detailed report
func CheckAllPods(checkAllCtx context.Context, pods map[string]*GoldpingerPod) *models.CheckAllResults {
	result := models.CheckAllResults{Responses: make(map[string]models.CheckAllPodResult)}

	ch := make(chan CheckServicePodsResult, len(pods))
	wg := sync.WaitGroup{}
	wg.Add(len(pods))

	for _, pod := range pods {
		go func(pod *GoldpingerPod) {
			// logger
			logger := zap.L().With(
				zap.String("op", "check"),
				zap.String("name", pod.Name),
				zap.String("hostIP", pod.HostIP),
				zap.String("podIP", pod.PodIP),
			)

			// stats
			CountCall("made", "check")
			timer := GetLabeledPeersCallsTimer("check", pod.HostIP, pod.PodIP)

			// setup
			var channelResult CheckServicePodsResult
			channelResult.podName = pod.Name
			channelResult.hostIPv4.UnmarshalText([]byte(pod.HostIP))
			channelResult.podIPv4.UnmarshalText([]byte(pod.PodIP))
			client, err := getClient(pickPodHostIP(pod.PodIP, pod.HostIP))
			OK := false

			if err != nil {
				logger.Warn("Couldn't get a client for Check", zap.Error(err))
				channelResult.checkAllPodResult = models.CheckAllPodResult{
					OK:     &OK,
					PodIP:  channelResult.podIPv4,
					HostIP: channelResult.hostIPv4,
					Error:  err.Error(),
				}
				CountError("checkAll")
			} else {
				checkCtx, cancel := context.WithTimeout(
					checkAllCtx,
					GoldpingerConfig.CheckTimeout,
				)
				defer cancel()

				params := operations.NewCheckServicePodsParamsWithContext(checkCtx)
				resp, err := client.Operations.CheckServicePods(params)
				OK = (err == nil)
				if OK {
					logger.Debug("Check Ok")
					channelResult.checkAllPodResult = models.CheckAllPodResult{
						OK:       &OK,
						PodIP:    channelResult.podIPv4,
						HostIP:   channelResult.hostIPv4,
						Response: resp.Payload,
					}
					timer.ObserveDuration()
				} else {
					logger.Warn("Check returned error", zap.Error(err))
					channelResult.checkAllPodResult = models.CheckAllPodResult{
						OK:     &OK,
						PodIP:  channelResult.podIPv4,
						HostIP: channelResult.hostIPv4,
						Error:  err.Error(),
					}
					CountError("checkAll")
				}
			}

			ch <- channelResult
			wg.Done()
		}(pod)
	}
	wg.Wait()
	close(ch)

	for response := range ch {
		result.Responses[response.podName] = response.checkAllPodResult
		result.Hosts = append(result.Hosts, &models.CheckAllResultsHostsItems0{
			PodName: response.podName,
			HostIP:  response.hostIPv4,
			PodIP:   response.podIPv4,
		})
		if response.checkAllPodResult.Response != nil &&
			response.checkAllPodResult.Response.ProbeResults != nil {
			if result.ProbeResults == nil {
				result.ProbeResults = make(map[string]models.ProbeResults)
			}
			for host := range response.checkAllPodResult.Response.ProbeResults {
				if result.ProbeResults[host] == nil {
					result.ProbeResults[host] = make(map[string][]models.ProbeResult)
				}
				result.ProbeResults[host][response.podName] = response.checkAllPodResult.Response.ProbeResults[host]
			}
		}
	}
	return &result
}

// HealthCheck returns a simple 200 OK response to verify the API is up
func HealthCheck() *models.HealthCheckResults {
	ok := true
	start := time.Now()
	result := models.HealthCheckResults{
		OK:          &ok,
		DurationNs:  time.Since(start).Nanoseconds(),
		GeneratedAt: strfmt.DateTime(start),
	}
	return &result
}

func getClient(hostIP string) (*apiclient.Goldpinger, error) {
	if hostIP == "" {
		return nil, errors.New("Host or pod IP empty, can't make a call")
	}
	host := net.JoinHostPort(hostIP, strconv.Itoa(GoldpingerConfig.Port))
	transport := httptransport.New(host, "", nil)
	client := apiclient.New(transport, strfmt.Default)
	apiclient.Default.SetTransport(transport)

	return client, nil
}
