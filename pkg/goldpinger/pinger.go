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
	"time"

	"go.uber.org/zap"

	apiclient "github.com/bloomberg/goldpinger/v3/pkg/client"
	"github.com/bloomberg/goldpinger/v3/pkg/client/operations"
	"github.com/bloomberg/goldpinger/v3/pkg/models"
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Pinger contains all the info needed by a goroutine to continuously ping a pod
type Pinger struct {
	pod         *GoldpingerPod
	client      *apiclient.Goldpinger
	timeout     time.Duration
	histogram   prometheus.Observer
	hostIPv4    strfmt.IPv4
	podIPv4     strfmt.IPv4
	resultsChan chan<- PingAllPodsResult
	stopChan    chan struct{}
	logger      *zap.Logger
}

// NewPinger constructs and returns a Pinger object responsible for pinging a single
// goldpinger pod
func NewPinger(pod *GoldpingerPod, resultsChan chan<- PingAllPodsResult) *Pinger {
	p := Pinger{
		pod:         pod,
		timeout:     time.Duration(GoldpingerConfig.PingTimeoutMs) * time.Millisecond,
		resultsChan: resultsChan,
		stopChan:    make(chan struct{}),

		histogram: goldpingerResponseTimePeersHistogram.WithLabelValues(
			GoldpingerConfig.Hostname,
			"ping",
			pod.HostIP,
			pod.PodIP,
		),

		logger: zap.L().With(
			zap.String("op", "pinger"),
			zap.String("name", pod.Name),
			zap.String("hostIP", pod.HostIP),
			zap.String("podIP", pod.PodIP),
		),
	}

	// Initialize the host/pod IPv4
	p.hostIPv4.UnmarshalText([]byte(pod.HostIP))
	p.podIPv4.UnmarshalText([]byte(pod.PodIP))

	return &p
}

// getClient returns a client that can be used to ping the given pod
// On error, it returns a static result
func (p *Pinger) getClient() (*apiclient.Goldpinger, error) {
	if p.client != nil {
		return p.client, nil
	}

	client, err := getClient(pickPodHostIP(p.pod.PodIP, p.pod.HostIP))
	if err != nil {
		p.logger.Warn("Could not get client", zap.Error(err))
		OK := false
		p.resultsChan <- PingAllPodsResult{
			podName: p.pod.Name,
			podResult: models.PodResult{
				PingTime:       strfmt.DateTime(time.Now()),
				PodIP:          p.podIPv4,
				HostIP:         p.hostIPv4,
				OK:             &OK,
				Error:          err.Error(),
				StatusCode:     500,
				ResponseTimeMs: 0,
			},
		}
		return nil, err
	}
	p.client = client
	return p.client, nil
}

// Ping makes a single ping request to the given pod
func (p *Pinger) Ping() {
	client, err := p.getClient()
	if err != nil {
		return
	}

	CountCall("made", "ping")
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	params := operations.NewPingParamsWithContext(ctx)
	resp, err := client.Operations.Ping(params)
	responseTime := time.Since(start)
	responseTimeMs := responseTime.Nanoseconds() / int64(time.Millisecond)
	p.histogram.Observe(responseTime.Seconds())

	OK := (err == nil)
	if OK {
		p.resultsChan <- PingAllPodsResult{
			podName: p.pod.Name,
			podResult: models.PodResult{
				PingTime:       strfmt.DateTime(start),
				PodIP:          p.podIPv4,
				HostIP:         p.hostIPv4,
				OK:             &OK,
				Response:       resp.Payload,
				StatusCode:     200,
				ResponseTimeMs: responseTimeMs,
			},
		}
		p.logger.Debug("Success pinging pod", zap.Duration("responseTime", responseTime))
	} else {
		p.resultsChan <- PingAllPodsResult{
			podName: p.pod.Name,
			podResult: models.PodResult{
				PingTime:       strfmt.DateTime(start),
				PodIP:          p.podIPv4,
				HostIP:         p.hostIPv4,
				OK:             &OK,
				Error:          err.Error(),
				StatusCode:     504,
				ResponseTimeMs: responseTimeMs,
			},
		}
		p.logger.Warn("Ping returned error", zap.Duration("responseTime", responseTime), zap.Error(err))
		CountError("ping")
	}
}

// PingContinuously continuously pings the given pod with a delay between
// `period` and `period + jitterFactor * period`
func (p *Pinger) PingContinuously(initialWait time.Duration, period time.Duration, jitterFactor float64) {
	p.logger.Info(
		"Starting pinger",
		zap.Duration("period", period),
		zap.Duration("initialWait", initialWait),
		zap.Float64("jitterFactor", jitterFactor),
	)

	timer := time.NewTimer(initialWait)

	select {
	case <-timer.C:
		wait.JitterUntil(p.Ping, period, jitterFactor, false, p.stopChan)
	case <-p.stopChan:
		// Do nothing
	}
	// We are done, send a message on the results channel to delete this
	p.resultsChan <- PingAllPodsResult{podName: p.pod.Name, deleted: true}
}
