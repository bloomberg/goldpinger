package goldpinger

import (
	"context"
	"time"

	"go.uber.org/zap"

	apiclient "github.com/bloomberg/goldpinger/v3/pkg/client"
	"github.com/bloomberg/goldpinger/v3/pkg/client/operations"
	"github.com/bloomberg/goldpinger/v3/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Pinger contains all the info needed by a goroutine to continuously ping a pod
type Pinger struct {
	pod         *GoldpingerPod
	client      *apiclient.Goldpinger
	timeout     time.Duration
	histogram   prometheus.Observer
	result      PingAllPodsResult
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

	// Initialize the result
	p.result.hostIPv4.UnmarshalText([]byte(pod.HostIP))
	p.result.podIPv4.UnmarshalText([]byte(pod.PodIP))

	// Get a client for pinging the given pod
	// On error, create a static pod result that does nothing
	client, err := getClient(pickPodHostIP(pod.PodIP, pod.HostIP))
	if err == nil {
		p.client = client
	} else {
		OK := false
		p.client = nil
		p.result.podResult = models.PodResult{HostIP: p.result.hostIPv4, OK: &OK, Error: err.Error(), StatusCode: 500, ResponseTimeMs: 0}
	}
	return &p
}

// Ping makes a single ping request to the given pod
func (p *Pinger) Ping() {
	CountCall("made", "ping")
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	params := operations.NewPingParamsWithContext(ctx)
	resp, err := p.client.Operations.Ping(params)
	responseTime := time.Since(start)
	responseTimeMs := responseTime.Nanoseconds() / int64(time.Millisecond)
	p.histogram.Observe(responseTime.Seconds())

	OK := (err == nil)
	if OK {
		p.result.podResult = models.PodResult{
			PodIP:          p.result.podIPv4,
			HostIP:         p.result.hostIPv4,
			OK:             &OK,
			Response:       resp.Payload,
			StatusCode:     200,
			ResponseTimeMs: responseTimeMs,
		}
		p.logger.Debug("Success pinging pod", zap.Duration("responseTime", responseTime))
	} else {
		p.result.podResult = models.PodResult{
			PodIP:          p.result.podIPv4,
			HostIP:         p.result.hostIPv4,
			OK:             &OK,
			Error:          err.Error(),
			StatusCode:     504,
			ResponseTimeMs: responseTimeMs,
		}
		p.logger.Warn("Ping returned error", zap.Duration("responseTime", responseTime), zap.Error(err))
		CountError("ping")
	}
	p.resultsChan <- p.result
}

// PingContinuously continuously pings the given pod with a delay between
// `period` and `period + jitterFactor * period`
func (p *Pinger) PingContinuously(period time.Duration, jitterFactor float64) {
	wait.JitterUntil(p.Ping, period, jitterFactor, false, p.stopChan)
}
