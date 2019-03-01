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
	"github.com/cespare/xxhash"
	rendezvous "github.com/stuartnelson3/go-rendezvous"
)

// PodSelecter selects the result of getPods() down to count instances
// according to a rendezvous hash.
type PodSelecter struct {
	count   uint
	podIP   string
	getPods func() map[string]string
}

// NewPodSelecter creates a new PodSelecter struct.
func NewPodSelecter(count uint, podIP string, getPods func() map[string]string) *PodSelecter {
	if podIP == "" {
		// If podIP is blank, then we can't use the rendezvous hash to
		// assign the IP correctly. Setting count=0 will force all pods
		// to be pinged.
		count = 0
	}
	return &PodSelecter{
		count:   count,
		podIP:   podIP,
		getPods: getPods,
	}
}

// SelectPods returns a map of pods filtered according to its configuration.
func (p *PodSelecter) SelectPods() map[string]string {
	allPods := p.getPods()
	if p.count == 0 || p.count >= uint(len(allPods)) {
		return allPods
	}
	rzv := rendezvous.New([]string{}, rendezvous.Hasher(xxhash.Sum64String))
	for podIP := range allPods {
		rzv.Add(podIP)
	}
	matches := rzv.LookupN(p.podIP, p.count)
	toPing := make(map[string]string)
	for _, podIP := range matches {
		toPing[podIP] = allPods[podIP]
	}
	return toPing
}
