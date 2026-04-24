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
	"io/ioutil"

	"go.uber.org/zap"
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
	if ipMatchesConfig(p.Status.HostIP) {
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

	var hostIP string
	for _, addr := range node.Status.Addresses {
		if (addr.Type == v1.NodeInternalIP || addr.Type == v1.NodeExternalIP) &&
			ipMatchesConfig(addr.Address) {
			hostIP = addr.Address
		}
	}
	nodeIPMap[p.Spec.NodeName] = hostIP
	return hostIP
}

// getPodIP will get an IPv6 IP from PodIPs if the UseIPv6 config is set, otherwise just return the object PodIP
func getPodIP(p v1.Pod) string {
	if ipMatchesConfig(p.Status.PodIP) {
		return p.Status.PodIP
	}

	var podIP string
	if p.Status.PodIPs != nil {
		for _, ip := range p.Status.PodIPs {
			if ipMatchesConfig(ip.IP) {
				podIP = ip.IP
			}
		}
	}
	return podIP
}

func getPodNodeName(p v1.Pod) string {
	if GoldpingerConfig.DisplayNodeName {
		return p.Spec.NodeName
	}

	return p.Name
}

// isReady is equivalent to k8s.io/kubectl/pkg/util/podutils.IsPodReady" but that brings us up a go version
// if we don't mind those dependencies could replace.
// it also checks DeletionTimestamp kubelet waits on next probe to set ready condition (endpoint slices remove immediately)
func isReady(pod *v1.Pod) bool {
	if pod.DeletionTimestamp != nil {
		return false
	}

	for _, cond := range pod.Status.Conditions {
		if cond.Type == v1.PodReady && cond.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

// GetAllPods returns a mapping from a pod name to a pointer to a GoldpingerPod(s)
func GetAllPods() map[string]*GoldpingerPod {
	timer := GetLabeledKubernetesCallsTimer()
	listOpts := metav1.ListOptions{
		ResourceVersion: "0",

		LabelSelector: GoldpingerConfig.LabelSelector,
		FieldSelector: "status.phase=Running", // only select Running pods, otherwise we will get them before they have IPs
	}
	pods, err := GoldpingerConfig.KubernetesClient.CoreV1().Pods(*GoldpingerConfig.Namespace).List(context.TODO(), listOpts)
	if err != nil {
		zap.L().Error("Error getting pods for selector", zap.String("selector", GoldpingerConfig.LabelSelector), zap.Error(err))
		CountError("kubernetes_api")
	} else {
		timer.ObserveDuration()
	}

	podMap := make(map[string]*GoldpingerPod)
	for _, pod := range pods.Items {
		if GoldpingerConfig.UseReadiness {
			if !isReady(&pod) {
				continue
			}
		}

		podMap[pod.Name] = &GoldpingerPod{
			Name:   getPodNodeName(pod),
			PodIP:  getPodIP(pod),
			HostIP: getHostIP(pod),
		}
	}
	return podMap
}

// ipMatchesConfig checks if the input IP family matches the first entry in the IPVersions config.
// TODO update to check all config versions to support dual-stack pinging.
func ipMatchesConfig(ip string) bool {
	ipFamily := getIPFamily(ip)
	return GoldpingerConfig.IPVersions[0] == string(ipFamily)
}

// getIPFamily returns the IP family of the input IP.
// Possible values are 4 and 6.
func getIPFamily(ip string) k8snet.IPFamily {
	if k8snet.IsIPv4String(ip) {
		return k8snet.IPv4
	}
	if k8snet.IsIPv6String(ip) {
		return k8snet.IPv6
	}
	zap.L().Error("Error determining IP family", zap.String("IP", ip))
	return ""
}
