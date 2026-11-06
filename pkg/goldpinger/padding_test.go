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
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bloomberg/goldpinger/v3/pkg/models"
	dto "github.com/prometheus/client_model/go"
)

const (
	pingPath        = "/ping"
	testPadSize     = 8192
	testPingTimeout = 500 * time.Millisecond
)

func TestGetPadding(t *testing.T) {
	tests := []struct {
		name string
		size int32
		want int
	}{
		{"Negative", -1, 0},
		{"Zero", 0, 0},
		{"One", 1, 1},
		{"Typical", testPadSize, testPadSize},
		{"Max", maxPingPadSize, maxPingPadSize},
		{"OverMax", maxPingPadSize + 1, maxPingPadSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPadding(tt.size)
			if len(got) != tt.want {
				t.Fatalf("GetPadding(%d) returned %d bytes, want %d", tt.size, len(got), tt.want)
			}
			// The padding goes into a JSON string verbatim: it must not
			// contain anything that would need escaping and change its size.
			if strings.ContainsAny(got, "\"\\") {
				t.Errorf("GetPadding(%d) contains characters that need JSON escaping", tt.size)
			}
		})
	}
}

// pingHandler answers GET /ping like a goldpinger peer. If honorPad is true
// it returns as much padding as asked for, otherwise it ignores the pad
// parameter like a peer running an older version. If stallPadded is true, it
// never answers padded pings in time, like a peer behind a PMTU black hole.
func pingHandler(t *testing.T, honorPad, stallPadded bool) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pingPath {
			t.Errorf("peer received unexpected request for %q, want %q", r.URL.Path, pingPath)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		result := models.PingResults{}
		if pad := r.URL.Query().Get("pad"); pad != "" {
			if stallPadded {
				select {
				case <-r.Context().Done():
				case <-time.After(2 * testPingTimeout):
				}
				return
			}
			if honorPad {
				n, err := strconv.Atoi(pad)
				if err != nil {
					t.Errorf("peer received invalid pad %q: %v", pad, err)
				}
				result.Padding = GetPadding(int32(n))
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			t.Errorf("peer could not write response body: %v", err)
		}
	})
}

func TestPingPadded(t *testing.T) {
	tests := []struct {
		name        string
		honorPad    bool
		stallPadded bool
		wantOK      float64
	}{
		{"PeerReturnsPadding", true, false, 1},
		{"PeerIgnoresPad", false, false, 0},
		{"PaddedResponseLost", true, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pods := startPeer(t, pingHandler(t, tt.honorPad, tt.stallPadded))
			GoldpingerConfig.PingTimeout = testPingTimeout
			GoldpingerConfig.PingPadSize = testPadSize

			resultsChan := make(chan PingAllPodsResult, 1)
			p := NewPinger(pods[peerPodName], resultsChan)
			client, err := p.getClient()
			if err != nil {
				t.Fatalf("could not get client: %v", err)
			}

			p.pingPadded(client)

			got := paddedPingOK(t)
			if got != tt.wantOK {
				t.Errorf("goldpinger_peers_padded_ping_ok = %v, want %v", got, tt.wantOK)
			}
		})
	}
}

// TestPingWithPadding verifies that a peer which answers regular pings but
// drops padded ones is still reported as reachable, and the padding failure
// shows up only in the padded ping metric.
func TestPingWithPadding(t *testing.T) {
	pods := startPeer(t, pingHandler(t, true, true))
	GoldpingerConfig.PingTimeout = testPingTimeout
	GoldpingerConfig.PingPadSize = testPadSize

	resultsChan := make(chan PingAllPodsResult, 1)
	p := NewPinger(pods[peerPodName], resultsChan)
	p.Ping()

	result := <-resultsChan
	if result.podResult.OK == nil || !*result.podResult.OK {
		t.Errorf("regular ping reported peer as unreachable: %+v", result.podResult)
	}
	got := paddedPingOK(t)
	if got != 0 {
		t.Errorf("goldpinger_peers_padded_ping_ok = %v, want 0", got)
	}
}

// paddedPingOK returns the padded ping gauge of the peer under test.
func paddedPingOK(t *testing.T) float64 {
	t.Helper()
	var d dto.Metric
	if err := goldpingerPeersPaddedPingOK.WithLabelValues(testHostname, peerHostIP, peerLoopbackIP).Write(&d); err != nil {
		t.Fatalf("could not read padded ping gauge: %v", err)
	}
	return d.GetGauge().GetValue()
}
