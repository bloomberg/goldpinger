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
	"log"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getNamespace() string {
	b, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Println("Unable to determine namespace: ", err.Error())
	}
	namespace := string(b)
	return namespace
}

func GetAllPods() map[string]string {
	timer := GetLabeledKubernetesCallsTimer()
	namespace := getNamespace()
	pods, err := GoldpingerConfig.KubernetesClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: "app=goldpinger"})
	if err != nil {
		log.Println("Error getting pods for selector: ", err.Error())
		CountError("kubernetes_api")
	} else {
		timer.ObserveDuration()
	}

	var podsreturn = make(map[string]string)
	for _, pod := range pods.Items {
		podsreturn[pod.Status.PodIP] = pod.Status.HostIP
	}
	return podsreturn
}
