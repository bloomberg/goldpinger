package goldpinger

import (
	"log"
	"time"

	apiclient "github.com/bloomberg/goldpinger/pkg/client"
	"github.com/bloomberg/goldpinger/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Pinger contains all the info needed by a goroutine to continuously ping a pod
type Pinger struct {
	podIP       string
	hostIP      string
	client      *apiclient.Goldpinger
	timer       *prometheus.Timer
	histogram   prometheus.Observer
	result      PingAllPodsResult
	resultsChan chan<- PingAllPodsResult
	stopChan    chan struct{}
}

// NewPinger constructs and returns a Pinger object responsible for pinging a single
// goldpinger pod
func NewPinger(podIP string, hostIP string, resultsChan chan<- PingAllPodsResult) *Pinger {
	p := Pinger{
		podIP:       podIP,
		hostIP:      hostIP,
		resultsChan: resultsChan,
		stopChan:    make(chan struct{}),

		histogram: goldpingerResponseTimePeersHistogram.WithLabelValues(
			GoldpingerConfig.Hostname,
			"ping",
			hostIP,
			podIP,
		),
	}

	// Initialize the result
	p.result.hostIPv4.UnmarshalText([]byte(hostIP))
	p.result.podIP = podIP

	// Get a client for pinging the given pod
	// On error, create a static pod result that does nothing
	client, err := getClient(pickPodHostIP(podIP, hostIP))
	if err == nil {
		p.client = client
	} else {
		OK := false
		p.client = nil
		p.result.podResult = models.PodResult{HostIP: p.result.hostIPv4, OK: &OK, Error: err.Error(), StatusCode: 500, ResponseTimeMs: 0}
		p.result.podIP = hostIP
	}
	return &p
}

// Ping makes a single ping request to the given pod
func (p *Pinger) Ping() {
	CountCall("made", "ping")
	start := time.Now()

	resp, err := p.client.Operations.Ping(nil)
	responseTime := time.Since(start)
	responseTimeMs := responseTime.Nanoseconds() / int64(time.Millisecond)
	p.histogram.Observe(responseTime.Seconds())

	OK := (err == nil)
	if OK {
		p.result.podResult = models.PodResult{HostIP: p.result.hostIPv4, OK: &OK, Response: resp.Payload, StatusCode: 200, ResponseTimeMs: responseTimeMs}
		log.Printf("Success pinging pod: %s, host: %s, resp: %+v, response time: %+v", p.podIP, p.hostIP, resp.Payload, responseTime)
	} else {
		p.result.podResult = models.PodResult{HostIP: p.result.hostIPv4, OK: &OK, Error: err.Error(), StatusCode: 504, ResponseTimeMs: responseTimeMs}
		log.Printf("Error pinging pod: %s, host: %s, err: %+v, response time: %+v", p.podIP, p.hostIP, err, responseTime)
		CountError("ping")
	}
	p.resultsChan <- p.result
}

// PingContinuously continuously pings the given pod with a delay between
// `period` and `period + jitterFactor * period`
func (p *Pinger) PingContinuously(period time.Duration, jitterFactor float64) {
	wait.JitterUntil(p.Ping, period, jitterFactor, false, p.stopChan)
}
