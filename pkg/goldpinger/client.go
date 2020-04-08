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
	"fmt"
	"net"
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
	if len(GoldpingerConfig.DnsHosts) > 0 {
		final.DNSResults = *checkDNS()
	}
	return &final
}

// CheckNeighboursNeighbours queries the kubernetes API server for all other goldpinger
// pods then calls Check() on each one
func CheckNeighboursNeighbours(ctx context.Context) *models.CheckAllResults {
	return CheckAllPods(ctx, SelectPods())
}

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

func checkDNS() *models.DNSResults {
	results := models.DNSResults{}
	for _, host := range GoldpingerConfig.DnsHosts {

		var dnsResult models.DNSResult

		start := time.Now()
		_, err := net.LookupIP(host)
		if err != nil {
			dnsResult.Error = err.Error()
			CountDnsError(host)
		}
		dnsResult.ResponseTimeMs = time.Since(start).Nanoseconds() / int64(time.Millisecond)
		results[host] = dnsResult
	}
	return &results
}

type CheckServicePodsResult struct {
	podName           string
	checkAllPodResult models.CheckAllPodResult
	hostIPv4          strfmt.IPv4
	podIPv4           strfmt.IPv4
}

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
					time.Duration(GoldpingerConfig.CheckTimeoutMs)*time.Millisecond,
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
			response.checkAllPodResult.Response.DNSResults != nil {
			if result.DNSResults == nil {
				result.DNSResults = make(map[string]models.DNSResults)
			}
			for host := range response.checkAllPodResult.Response.DNSResults {
				if result.DNSResults[host] == nil {
					result.DNSResults[host] = make(map[string]models.DNSResult)
				}
				result.DNSResults[host][response.podName] = response.checkAllPodResult.Response.DNSResults[host]
			}
		}
	}
	return &result
}

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
	host := fmt.Sprintf("%s:%d", hostIP, GoldpingerConfig.Port)
	transport := httptransport.New(host, "", nil)
	client := apiclient.New(transport, strfmt.Default)
	apiclient.Default.SetTransport(transport)

	return client, nil
}
