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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/bloomberg/goldpinger/v3/pkg/models"
	"github.com/go-openapi/strfmt"
)

const (
	// peerPodName is the name goldpinger discovers for the peer under test.
	peerPodName = "goldpinger-peer"
	// peerHostIP is the host IP goldpinger discovers for the peer under test.
	// The peer is dialled over loopback, but is reported to live on this node.
	peerHostIP = "10.0.0.1"
	// peerLoopbackIP is the address the peer's HTTP server actually listens on.
	peerLoopbackIP = "127.0.0.1"
	// checkPath is the path the generated client calls for a /check.
	checkPath = "/check"

	testCheckTimeout    = 5 * time.Second
	testCheckAllTimeout = 10 * time.Second
	testHostname        = "goldpinger-under-test"

	// floodedPodResults is the number of pod results a hostile peer crams into
	// a single /check response. Any value greater than the number of pods we
	// discovered ourselves is enough to walk off the end of the expected list.
	floodedPodResults = 5000
)

// startPeer runs a real HTTP server answering GET /check with the given
// handler and points GoldpingerConfig at its port, so the generated swagger
// client dials it over a real connection. It returns the single-pod peer set
// that checkClusterHealth should be given.
func startPeer(t *testing.T, handler http.Handler) map[string]*GoldpingerPod {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("could not parse peer server URL %q: %v", server.URL, err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("could not parse peer server port from %q: %v", server.URL, err)
	}

	originalConfig := GoldpingerConfig
	t.Cleanup(func() { GoldpingerConfig = originalConfig })
	GoldpingerConfig.Port = port
	GoldpingerConfig.UseHostIP = false
	GoldpingerConfig.Hostname = testHostname
	GoldpingerConfig.CheckTimeout = testCheckTimeout
	GoldpingerConfig.CheckAllTimeout = testCheckAllTimeout

	// Checking a peer registers per-peer label sets on package-global
	// collectors. Drop them again while the config they were labelled with is
	// still in place, so we don't leak label sets into other tests. Registered
	// after the config restore above so cleanups run in that order.
	t.Cleanup(func() { DeletePeerMetrics(peerHostIP, peerLoopbackIP) })

	return map[string]*GoldpingerPod{
		peerPodName: {Name: peerPodName, PodIP: peerLoopbackIP, HostIP: peerHostIP},
	}
}

// checkHandler serves body as the JSON response to GET /check, and fails the
// test if the client asks for anything else.
func checkHandler(t *testing.T, body []byte) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != checkPath {
			t.Errorf("peer received unexpected request for %q, want %q", r.URL.Path, checkPath)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(body); err != nil {
			t.Errorf("peer could not write response body: %v", err)
		}
	})
}

// peerCheckResults builds a /check response body listing one pod result per
// host IP given, as an honest peer would report its own neighbours.
func peerCheckResults(t *testing.T, hostIPs ...string) []byte {
	t.Helper()

	ok := true
	results := models.CheckResults{PodResults: make(map[string]models.PodResult, len(hostIPs))}
	for index, hostIP := range hostIPs {
		results.PodResults[fmt.Sprintf("pod-%d", index)] = models.PodResult{
			OK:         &ok,
			HostIP:     strfmt.IPv4(hostIP),
			PodIP:      strfmt.IPv4(hostIP),
			StatusCode: http.StatusOK,
		}
	}

	body, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("could not marshal peer check results: %v", err)
	}
	return body
}

// syntheticHostIPs generates count distinct host IPs, for peers that report
// far more nodes than actually exist.
func syntheticHostIPs(count int) []string {
	hostIPs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		hostIPs = append(hostIPs, fmt.Sprintf("10.%d.%d.%d", 1+i/65536, (i/256)%256, i%256))
	}
	return hostIPs
}

// TestCheckClusterHealthWithHostilePeer covers /cluster_health aggregation when
// the single discovered peer returns crafted results. A workload that can join
// the peer set controls this body entirely, so no shape of it may panic the
// process (CWE-248) — it may only mark the cluster not OK.
func TestCheckClusterHealthWithHostilePeer(t *testing.T) {
	tests := []struct {
		name string
		// body is the raw bytes the peer serves for GET /check.
		body []byte
		// wantOK is the expected aggregated cluster verdict.
		wantOK bool
	}{
		{
			name:   "peer reports exactly the expected node",
			body:   peerCheckResults(t, peerHostIP),
			wantOK: true,
		},
		{
			name:   "peer reports extra pod results",
			body:   peerCheckResults(t, peerHostIP, "10.0.0.2", "10.0.0.3"),
			wantOK: false,
		},
		{
			name:   "peer reports no pod results",
			body:   peerCheckResults(t),
			wantOK: false,
		},
		{
			name:   "peer reports a node we never discovered",
			body:   peerCheckResults(t, "10.9.9.9"),
			wantOK: false,
		},
		{
			name:   "peer floods pod results",
			body:   peerCheckResults(t, syntheticHostIPs(floodedPodResults)...),
			wantOK: false,
		},
		{
			name:   "peer returns a null body",
			body:   []byte("null"),
			wantOK: false,
		},
		{
			name:   "peer returns an empty object",
			body:   []byte("{}"),
			wantOK: false,
		},
		{
			name:   "peer returns pod results with empty host IPs",
			body:   []byte(`{"podResults":{"a":{},"b":{}}}`),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pods := startPeer(t, checkHandler(t, tt.body))

			// A panic anywhere in here fails the test: that is the bug under test.
			output := checkClusterHealth(context.Background(), pods)

			if output.OK != tt.wantOK {
				t.Errorf("cluster OK = %v, want %v (healthy=%v unhealthy=%v)",
					output.OK, tt.wantOK, output.NodesHealthy, output.NodesUnhealthy)
			}
			if output.NodesTotal != 1 {
				t.Errorf("NodesTotal = %d, want 1 — exactly one peer was discovered", output.NodesTotal)
			}
			// The peer answered with a 200, so it counts as a reachable node
			// regardless of how nonsensical its payload was.
			if len(output.NodesHealthy) != 1 || output.NodesHealthy[0] != peerHostIP {
				t.Errorf("NodesHealthy = %v, want [%s]", output.NodesHealthy, peerHostIP)
			}
			if len(output.NodesUnhealthy) != 0 {
				t.Errorf("NodesUnhealthy = %v, want []", output.NodesUnhealthy)
			}
		})
	}
}

// TestCheckClusterHealthWithUnreachablePeer covers peers that fail the /check
// call outright: they must be reported unhealthy rather than crashing or
// silently passing.
func TestCheckClusterHealthWithUnreachablePeer(t *testing.T) {
	tests := []struct {
		name    string
		handler http.Handler
	}{
		{
			name: "peer returns a server error",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}),
		},
		{
			name: "peer returns a body that is not JSON",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write([]byte("this is not json")); err != nil {
					t.Errorf("peer could not write response body: %v", err)
				}
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pods := startPeer(t, tt.handler)

			output := checkClusterHealth(context.Background(), pods)

			if output.OK {
				t.Errorf("cluster OK = true, want false for an unreachable peer")
			}
			if output.NodesTotal != 1 {
				t.Errorf("NodesTotal = %d, want 1", output.NodesTotal)
			}
			if len(output.NodesUnhealthy) != 1 || output.NodesUnhealthy[0] != peerHostIP {
				t.Errorf("NodesUnhealthy = %v, want [%s]", output.NodesUnhealthy, peerHostIP)
			}
			if len(output.NodesHealthy) != 0 {
				t.Errorf("NodesHealthy = %v, want []", output.NodesHealthy)
			}
		})
	}
}

// TestCheckClusterHealthWithNoPeers verifies the degenerate case: with nothing
// to talk to, we should at least not report the cluster as OK.
func TestCheckClusterHealthWithNoPeers(t *testing.T) {
	originalConfig := GoldpingerConfig
	t.Cleanup(func() { GoldpingerConfig = originalConfig })
	GoldpingerConfig.Hostname = testHostname
	GoldpingerConfig.CheckTimeout = testCheckTimeout
	GoldpingerConfig.CheckAllTimeout = testCheckAllTimeout

	output := checkClusterHealth(context.Background(), map[string]*GoldpingerPod{})

	if output.OK {
		t.Errorf("cluster OK = true, want false when no peers responded")
	}
	if output.NodesTotal != 0 {
		t.Errorf("NodesTotal = %d, want 0", output.NodesTotal)
	}
}
