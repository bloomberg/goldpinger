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

// SelectPods selects a set of pods from the results of GetAllPods
// depending on the count according to a rendezvous hash
func SelectPods() map[string]*GoldpingerPod {
	allPods := GetAllPods()
	if GoldpingerConfig.PingNumber <= 0 || int(GoldpingerConfig.PingNumber) >= len(allPods) {
		return allPods
	}

	rzv := rendezvous.New([]string{}, rendezvous.Hasher(xxhash.Sum64String))
	for podName := range allPods {
		rzv.Add(podName)
	}
	matches := rzv.LookupN(GoldpingerConfig.PodName, GoldpingerConfig.PingNumber)
	toPing := make(map[string]*GoldpingerPod)
	for _, podName := range matches {
		toPing[podName] = allPods[podName]
	}
	return toPing
}
