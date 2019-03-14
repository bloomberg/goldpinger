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
	Hostname         string `long:"hostname" description:"Hostname to use" env:"HOSTNAME"`
	PodIP            string `long:"pod-ip" description:"Pod IP to use" env:"POD_IP"`
	PingNumber       uint   `long:"ping-number" description:"Number of peers to ping. A value of 0 indicates all peers should be pinged." default:"0" env:"PING_NUMBER"`
	Port             int    `long:"client-port-override" description:"(for testing) use this port when calling other instances" env:"CLIENT_PORT_OVERRIDE"`
	UseHostIP        bool   `long:"use-host-ip" description:"When making the calls, use host ip (defaults to pod ip)" env:"USE_HOST_IP"`
	LabelSelector    string `long:"label-selector" description:"label selector to use to discover goldpinger pods in the cluster" env:"LABEL_SELECTOR" default:"app=goldpinger"`
	KubernetesClient *kubernetes.Clientset
	*PodSelecter
}{}
