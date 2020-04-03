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
	"k8s.io/client-go/kubernetes"
)

// GoldpingerConfig represents the configuration for goldpinger
var GoldpingerConfig = struct {
	StaticFilePath   string `long:"static-file-path" description:"Folder for serving static files" env:"STATIC_FILE_PATH"`
	KubeConfigPath   string `long:"kubeconfig" description:"Path to kubeconfig file" env:"KUBECONFIG"`
	RefreshInterval  int    `long:"refresh-interval" description:"If > 0, will create a thread and collect stats every n seconds" env:"REFRESH_INTERVAL" default:"30"`
	JitterFactor     float64 `long:"jitter-factor" description:"The amount of jitter to add while pinging clients" env:"JITTER_FACTOR" default:"0.05"`
	Hostname         string `long:"hostname" description:"Hostname to use" env:"HOSTNAME"`
	PodIP            string `long:"pod-ip" description:"Pod IP to use" env:"POD_IP"`
	PodName          string `long:"pod-name" description:"The name of this pod - used to select --ping-number of pods using rendezvous hashing" env:"POD_NAME"`
	PingNumber       uint   `long:"ping-number" description:"Number of peers to ping. A value of 0 indicates all peers should be pinged." default:"0" env:"PING_NUMBER"`
	Port             int    `long:"client-port-override" description:"(for testing) use this port when calling other instances" env:"CLIENT_PORT_OVERRIDE"`
	UseHostIP        bool   `long:"use-host-ip" description:"When making the calls, use host ip (defaults to pod ip)" env:"USE_HOST_IP"`
	LabelSelector    string `long:"label-selector" description:"label selector to use to discover goldpinger pods in the cluster" env:"LABEL_SELECTOR" default:"app=goldpinger"`
	KubernetesClient *kubernetes.Clientset

	DnsHosts []string `long:"host-to-resolve" description:"A host to attempt dns resolve on (space delimited)" env:"HOSTS_TO_RESOLVE" env-delim:" "`

	// Timeouts
	PingTimeoutMs     int64 `long:"ping-timeout-ms" description:"The timeout in milliseconds for a ping call to other goldpinger pods" env:"PING_TIMEOUT_MS" default:"300"`
	CheckTimeoutMs    int64 `long:"check-timeout-ms" description:"The timeout in milliseconds for a check call to other goldpinger pods" env:"CHECK_TIMEOUT_MS" default:"1000"`
	CheckAllTimeoutMs int64 `long:"check-all-timeout-ms" description:"The timeout in milliseconds for a check-all call to other goldpinger pods" env:"CHECK_ALL_TIMEOUT_MS" default:"5000"`
}{}
