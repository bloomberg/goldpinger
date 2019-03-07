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

package main

import (
	"log"
	"os"

	"github.com/go-openapi/loads"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/bloomberg/goldpinger/pkg/goldpinger"
	"github.com/bloomberg/goldpinger/pkg/restapi"
	"github.com/bloomberg/goldpinger/pkg/restapi/operations"
	flags "github.com/jessevdk/go-flags"
)

// these will be injected during build in build.sh script to allow printing
var (
	Version, Build string
)

func main() {

	log.Println("Goldpinger version:", Version, "build:", Build)

	// load embedded swagger file
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	// create new service API
	api := operations.NewGoldpingerAPI(swaggerSpec)
	server := restapi.NewServer(api)
	defer server.Shutdown()

	parser := flags.NewParser(server, flags.Default)
	parser.ShortDescription = "Goldpinger"
	parser.LongDescription = swaggerSpec.Spec().Info.Description

	// parse flags
	server.ConfigureFlags()
	for _, optsGroup := range api.CommandLineOptionsGroups {
		_, err := parser.AddGroup(optsGroup.ShortDescription, optsGroup.LongDescription, optsGroup.Options)
		if err != nil {
			log.Fatalln(err)
		}
	}

	if _, err := parser.Parse(); err != nil {
		code := 1
		if fe, ok := err.(*flags.Error); ok {
			if fe.Type == flags.ErrHelp {
				code = 0
			}
		}
		os.Exit(code)
	}

	// make a kubernetes client
	var config *rest.Config
	if goldpinger.GoldpingerConfig.KubeConfigPath == "" {
		log.Println("Kubeconfig not specified, trying to use in cluster config")
		config, err = rest.InClusterConfig()
	} else {
		log.Println("Kubeconfig specified in ", goldpinger.GoldpingerConfig.KubeConfigPath)
		config, err = clientcmd.BuildConfigFromFlags("", goldpinger.GoldpingerConfig.KubeConfigPath)
	}
	if err != nil {
		log.Fatalln("Error getting config ", err.Error())
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln("kubernetes.NewForConfig error ", err.Error())
	}
	goldpinger.GoldpingerConfig.KubernetesClient = clientset

	// Check if we have an override for the client, default to own port
	if goldpinger.GoldpingerConfig.Port == 0 {
		goldpinger.GoldpingerConfig.Port = server.Port
	}

	if goldpinger.GoldpingerConfig.PodIP == "" {
		log.Println("PodIP not set: pinging all pods")
	}
	if goldpinger.GoldpingerConfig.PingNumber == 0 {
		log.Println("--ping-number set to 0: pinging all pods")
	}
	goldpinger.GoldpingerConfig.PodSelecter = goldpinger.NewPodSelecter(goldpinger.GoldpingerConfig.PingNumber, goldpinger.GoldpingerConfig.PodIP, goldpinger.GetAllPods)

	server.ConfigureAPI()
	goldpinger.StartUpdater()

	log.Println("All good, starting serving the API")

	// serve API
	if err := server.Serve(); err != nil {
		log.Fatalln(err)
	}
}
