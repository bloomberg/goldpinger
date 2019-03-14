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
	"log"
	"time"
)

func StartUpdater() {
	if GoldpingerConfig.RefreshInterval <= 0 {
		log.Println("Not creating updater, period is 0")
		return
	}

	// start the updater
	go func() {
		for {
			results := PingAllPods(GoldpingerConfig.PodSelecter.SelectPods())
			var troublemakers []string
			for podIP, value := range results {
				if *value.OK != true {
					troublemakers = append(troublemakers, fmt.Sprintf("%s (%s)", podIP, value.HostIP.String()))
				}
			}
			if len(troublemakers) > 0 {
				log.Println("Updater ran into trouble with these peers: ", troublemakers)
			}
			time.Sleep(time.Duration(GoldpingerConfig.RefreshInterval) * time.Second)
		}
	}()
}
