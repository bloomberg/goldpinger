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
	"go.uber.org/zap"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8snet "k8s.io/utils/net"
)

var nodeIPMap = make(map[string]string)

// PodNamespace is the auto-detected namespace for this goldpinger pod
var PodNamespace = getPodNamespace()

// GoldpingerPod contains just the basic info needed to ping and keep track of a given goldpinger pod
type GoldpingerPod struct {
	Name   string // Name is the name of the pod
	PodIP  string // PodIP is the IP address of the pod
	HostIP string // HostIP is the IP address of the host where the pod lives
}

func getPodNamespace() string {
	b, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		zap.L().Warn("Unable to determine namespace", zap.Error(err))
		return ""
	}
	namespace := string(b)
	return namespace
}

// getHostIP gets the IP of the host where the pod is scheduled. If UseIPv6 is enabled then we need to check
// the node IPs since HostIP will only list the default IP version one.
func getHostIP(p v1.Pod) string {
	if !GoldpingerConfig.UseIPv6 || k8snet.IsIPv6String(p.Status.HostIP) {
		return p.Status.HostIP
	}

	if addr, ok := nodeIPMap[p.Spec.NodeName]; ok {
		return addr
	}

	timer := GetLabeledKubernetesCallsTimer()
	node, err := GoldpingerConfig.KubernetesClient.CoreV1().Nodes().Get(context.TODO(), p.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		zap.L().Error("error getting node", zap.Error(err))
		CountError("kubernetes_api")
		return p.Status.HostIP
	} else {
		timer.ObserveDuration()
	}

	result := p.Status.HostIP
	for _, addr := range node.Status.Addresses {
		if k8snet.IsIPv6String(addr.Address) {
			result = addr.Address
		}
	}
	nodeIPMap[p.Spec.NodeName] = result
	return result
}

// getPodIP will get an IPv6 IP from PodIPs if the UseIPv6 config is set, otherwise just return the object PodIP
func getPodIP(p v1.Pod) string {
	if !GoldpingerConfig.UseIPv6 {
		return p.Status.PodIP
	}

	var v6IP string
	if p.Status.PodIPs != nil {
		for _, ip := range p.Status.PodIPs {
			if k8snet.IsIPv6String(ip.IP) {
				v6IP = ip.IP
			}
		}
	}
	return v6IP
}

// GetAllPods returns a mapping from a pod name to a pointer to a GoldpingerPod(s)
func GetAllPods() map[string]*GoldpingerPod {
	timer := GetLabeledKubernetesCallsTimer()
	pods, err := GoldpingerConfig.KubernetesClient.CoreV1().Pods(*GoldpingerConfig.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: GoldpingerConfig.LabelSelector})
	if err != nil {
		zap.L().Error("Error getting pods for selector", zap.String("selector", GoldpingerConfig.LabelSelector), zap.Error(err))
		CountError("kubernetes_api")
	} else {
		timer.ObserveDuration()
	}

	var podMap = make(map[string]*GoldpingerPod)
	for _, pod := range pods.Items {
		podMap[pod.Name] = &GoldpingerPod{
			Name:   pod.Name,
			PodIP:  getPodIP(pod),
			HostIP: getHostIP(pod),
		}
	}
	return podMap
}
