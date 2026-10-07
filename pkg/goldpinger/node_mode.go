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
	"hash/fnv"
)

// NodeMode holds the determined mode for this node ("active" or "passive").
// Set once at startup, read thereafter.
var NodeMode string

const (
	NodeModeActive  = "active"
	NodeModePassive = "passive"
)

// DetermineNodeMode decides whether this node should be active or passive.
//
// Priority:
//  1. If hostname appears in activeNodes list -> active (explicit override)
//  2. If FNV-1a(hostname) % 100 < percentage -> active (hash-based)
//  3. Otherwise -> passive
//
// percentage is clamped to [1, 100]. An empty hostname is treated as active
// (fail-open: a misconfigured pod should not silently go passive).
func DetermineNodeMode(hostname string, percentage int, activeNodes []string) string {
	// Fail-open: if hostname is empty, be active to avoid silent failures
	if hostname == "" {
		return NodeModeActive
	}

	// Check explicit override list first
	for _, name := range activeNodes {
		if name == hostname {
			return NodeModeActive
		}
	}

	// Clamp percentage
	if percentage < 1 {
		percentage = 1
	}
	if percentage > 100 {
		percentage = 100
	}

	// 100% means all active, skip hashing
	if percentage >= 100 {
		return NodeModeActive
	}

	// FNV-1a hash of hostname
	bucket := NodeHashBucket(hostname)

	if bucket < uint32(percentage) {
		return NodeModeActive
	}

	return NodeModePassive
}

// NodeHashBucket returns the hash bucket (0-99) for a given hostname.
func NodeHashBucket(hostname string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(hostname))
	return h.Sum32() % 100
}

// IsActiveNode returns true if this node is in active mode.
func IsActiveNode() bool {
	return NodeMode == NodeModeActive
}
