package goldpinger

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// TestDeletePeerMetrics_CleansResponseTimeHistogram verifies that
// DeletePeerMetrics removes the response-time histogram label set for
// a destroyed peer. This prevents stale pod IPs from lingering in
// /metrics after rolling updates (see #167).
func TestDeletePeerMetrics_CleansResponseTimeHistogram(t *testing.T) {
	tests := []struct {
		name   string
		hostIP string
		podIP  string
	}{
		{"IPv4", "10.0.0.1", "10.0.0.2"},
		{"IPv6", "2001:db8::1", "2001:db8:1::2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origHostname := GoldpingerConfig.Hostname
			GoldpingerConfig.Hostname = "test-instance"
			defer func() { GoldpingerConfig.Hostname = origHostname }()

			// Simulate observations for both call_type values the histogram uses
			goldpingerResponseTimePeersHistogram.WithLabelValues(
				GoldpingerConfig.Hostname, "ping", tt.hostIP, tt.podIP,
			).Observe(0.005)
			goldpingerResponseTimePeersHistogram.WithLabelValues(
				GoldpingerConfig.Hostname, "check", tt.hostIP, tt.podIP,
			).Observe(0.010)

			if n := countMetrics(goldpingerResponseTimePeersHistogram); n != 2 {
				t.Fatalf("response time histogram has %d label sets before cleanup, want 2 — test setup is broken", n)
			}

			DeletePeerMetrics(tt.hostIP, tt.podIP)

			if n := countMetrics(goldpingerResponseTimePeersHistogram); n != 0 {
				t.Errorf("response time histogram still has %d label set(s) after DeletePeerMetrics", n)
			}
		})
	}
}

// TestDeletePeerMetrics_LeavesOtherPeersIntact verifies that pruning
// metrics for one peer does not affect a different peer's label set.
func TestDeletePeerMetrics_LeavesOtherPeersIntact(t *testing.T) {
	tests := []struct {
		name    string
		peerA   [2]string // {hostIP, podIP}
		peerB   [2]string
	}{
		{
			"IPv4",
			[2]string{"10.0.0.1", "10.0.0.2"},
			[2]string{"10.0.0.3", "10.0.0.4"},
		},
		{
			"IPv6",
			[2]string{"2001:db8::1", "2001:db8:1::2"},
			[2]string{"2001:db8::3", "2001:db8:2::4"},
		},
		{
			"MixedV4DeleteV6Survives",
			[2]string{"10.0.0.1", "10.0.0.2"},
			[2]string{"2001:db8::3", "2001:db8:2::4"},
		},
		{
			"MixedV6DeleteV4Survives",
			[2]string{"2001:db8::1", "2001:db8:1::2"},
			[2]string{"10.0.0.3", "10.0.0.4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origHostname := GoldpingerConfig.Hostname
			GoldpingerConfig.Hostname = "test-instance"
			defer func() { GoldpingerConfig.Hostname = origHostname }()

			// Peer A — observe both call types
			goldpingerResponseTimePeersHistogram.WithLabelValues(
				GoldpingerConfig.Hostname, "ping", tt.peerA[0], tt.peerA[1],
			).Observe(0.005)
			goldpingerResponseTimePeersHistogram.WithLabelValues(
				GoldpingerConfig.Hostname, "check", tt.peerA[0], tt.peerA[1],
			).Observe(0.006)
			SetPeerLossPct(tt.peerA[0], tt.peerA[1], 0)

			// Peer B — observe both call types
			goldpingerResponseTimePeersHistogram.WithLabelValues(
				GoldpingerConfig.Hostname, "ping", tt.peerB[0], tt.peerB[1],
			).Observe(0.010)
			goldpingerResponseTimePeersHistogram.WithLabelValues(
				GoldpingerConfig.Hostname, "check", tt.peerB[0], tt.peerB[1],
			).Observe(0.011)
			SetPeerLossPct(tt.peerB[0], tt.peerB[1], 1.5)

			// Delete peer A only
			DeletePeerMetrics(tt.peerA[0], tt.peerA[1])
			DeletePeerUDPMetrics(tt.peerA[0], tt.peerA[1])

			// Peer B's ping and check histogram entries should both survive
			if n := countMetrics(goldpingerResponseTimePeersHistogram); n != 2 {
				t.Errorf("response time histogram has %d label set(s), want 2 for peer B (ping+check)", n)
			}
			if countMetrics(goldpingerPeersLossPct) == 0 {
				t.Error("loss pct gauge lost all label sets — peer B should still exist")
			}

			// Clean up peer B so it doesn't leak into other tests
			goldpingerResponseTimePeersHistogram.DeleteLabelValues(
				GoldpingerConfig.Hostname, "ping", tt.peerB[0], tt.peerB[1],
			)
			goldpingerResponseTimePeersHistogram.DeleteLabelValues(
				GoldpingerConfig.Hostname, "check", tt.peerB[0], tt.peerB[1],
			)
			goldpingerPeersLossPct.DeleteLabelValues(
				GoldpingerConfig.Hostname, tt.peerB[0], tt.peerB[1],
			)
		})
	}
}

// TestDeletePeerUDPMetrics_CleansAllPerPeerMetrics verifies that
// DeletePeerUDPMetrics removes label sets from every per-peer UDP metric.
// If a new per-peer metric is added but not cleaned up in
// DeletePeerUDPMetrics, this test will fail.
func TestDeletePeerUDPMetrics_CleansAllPerPeerMetrics(t *testing.T) {
	tests := []struct {
		name   string
		hostIP string
		podIP  string
	}{
		{"IPv4", "10.0.0.1", "10.0.0.2"},
		{"IPv6", "2001:db8::1", "2001:db8:1::2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origHostname := GoldpingerConfig.Hostname
			origUseHostIP := GoldpingerConfig.UseHostIP
			GoldpingerConfig.Hostname = "test-instance"
			GoldpingerConfig.UseHostIP = false
			defer func() {
				GoldpingerConfig.Hostname = origHostname
				GoldpingerConfig.UseHostIP = origUseHostIP
			}()

			// Populate all per-peer UDP metrics so they have label values
			SetPeerLossPct(tt.hostIP, tt.podIP, 5.0)
			SetPeerHopCount(tt.hostIP, tt.podIP, 2)
			ObservePeerUDPRtt(tt.hostIP, tt.podIP, 0.001)
			CountUDPDuplicates(tt.hostIP, tt.podIP, 1)
			CountUDPOutOfOrder(tt.hostIP, tt.podIP, 1)
			CountUDPError(tt.podIP) // UseHostIP=false so target=podIP

			// Verify they exist before cleanup
			perPeerCollectors := map[string]prometheus.Collector{
				"goldpinger_peers_loss_pct":         goldpingerPeersLossPct,
				"goldpinger_peers_hop_count":        goldpingerPeersHopCount,
				"goldpinger_peers_udp_rtt_s":        goldpingerPeersUDPRtt,
				"goldpinger_udp_duplicates_total":   goldpingerUDPDuplicatesCounter,
				"goldpinger_udp_out_of_order_total": goldpingerUDPOutOfOrderCounter,
				"goldpinger_udp_errors_total":       goldpingerUDPErrorsCounter,
			}

			for name, collector := range perPeerCollectors {
				if countMetrics(collector) == 0 {
					t.Fatalf("metric %s has no label values before cleanup — test setup is broken", name)
				}
			}

			// Run cleanup
			DeletePeerUDPMetrics(tt.hostIP, tt.podIP)

			// Verify all per-peer metrics are cleaned up
			for name, collector := range perPeerCollectors {
				if n := countMetrics(collector); n != 0 {
					t.Errorf("metric %s still has %d label set(s) after DeletePeerUDPMetrics — add it to the cleanup function", name, n)
				}
			}
		})
	}
}

// countMetrics returns the number of metric families (label sets) for a collector.
func countMetrics(c prometheus.Collector) int {
	ch := make(chan prometheus.Metric, 100)
	c.Collect(ch)
	close(ch)
	count := 0
	for m := range ch {
		var d dto.Metric
		if err := m.Write(&d); err == nil {
			count++
		}
	}
	return count
}
