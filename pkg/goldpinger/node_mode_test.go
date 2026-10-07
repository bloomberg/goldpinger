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
	"fmt"
	"testing"
)

func TestDetermineNodeMode_Percentage100AlwaysActive(t *testing.T) {
	hostnames := []string{"node-1", "node-2", "worker-abc", "ip-10-0-1-5"}
	for _, h := range hostnames {
		mode := DetermineNodeMode(h, 100, nil)
		if mode != NodeModeActive {
			t.Errorf("Expected active for hostname %q at 100%%, got %s", h, mode)
		}
	}
}

func TestDetermineNodeMode_Percentage0ClampsTo1(t *testing.T) {
	// At 1%, most hostnames should be passive, but it shouldn't panic or error
	mode := DetermineNodeMode("test-node", 0, nil)
	if mode != NodeModeActive && mode != NodeModePassive {
		t.Errorf("Expected active or passive, got %s", mode)
	}
}

func TestDetermineNodeMode_PercentageAbove100ClampsTo100(t *testing.T) {
	mode := DetermineNodeMode("test-node", 150, nil)
	if mode != NodeModeActive {
		t.Errorf("Expected active for percentage > 100, got %s", mode)
	}
}

func TestDetermineNodeMode_ActiveNodesOverride(t *testing.T) {
	// Even at 1% where hash would likely be passive, explicit list wins
	activeNodes := []string{"forced-node"}
	mode := DetermineNodeMode("forced-node", 1, activeNodes)
	if mode != NodeModeActive {
		t.Errorf("Expected active for node in ACTIVE_NODES list, got %s", mode)
	}
}

func TestDetermineNodeMode_ActiveNodesNoMatchFallsToHash(t *testing.T) {
	activeNodes := []string{"other-node"}
	// At 100%, hash would make it active anyway
	mode := DetermineNodeMode("test-node", 100, activeNodes)
	if mode != NodeModeActive {
		t.Errorf("Expected active at 100%%, got %s", mode)
	}
}

func TestDetermineNodeMode_EmptyHostnameFailsOpen(t *testing.T) {
	mode := DetermineNodeMode("", 1, nil)
	if mode != NodeModeActive {
		t.Errorf("Expected active for empty hostname (fail-open), got %s", mode)
	}
}

func TestDetermineNodeMode_Deterministic(t *testing.T) {
	hostname := "consistent-node-name"
	first := DetermineNodeMode(hostname, 50, nil)
	for i := 0; i < 100; i++ {
		result := DetermineNodeMode(hostname, 50, nil)
		if result != first {
			t.Fatalf("Non-deterministic result on iteration %d: got %s, expected %s", i, result, first)
		}
	}
}

func TestDetermineNodeMode_DistributionAt50Percent(t *testing.T) {
	active := 0
	total := 10000
	for i := 0; i < total; i++ {
		hostname := fmt.Sprintf("node-%d", i)
		if DetermineNodeMode(hostname, 50, nil) == NodeModeActive {
			active++
		}
	}
	ratio := float64(active) / float64(total)
	if ratio < 0.40 || ratio > 0.60 {
		t.Errorf("Expected ~50%% active at 50%% setting, got %.1f%% (%d/%d)", ratio*100, active, total)
	}
}

func TestIsActiveNode(t *testing.T) {
	NodeMode = NodeModeActive
	if !IsActiveNode() {
		t.Error("Expected IsActiveNode() to return true when NodeMode is active")
	}
	NodeMode = NodeModePassive
	if IsActiveNode() {
		t.Error("Expected IsActiveNode() to return false when NodeMode is passive")
	}
}
