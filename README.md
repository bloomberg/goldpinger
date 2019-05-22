# Goldpinger [![Build Status](https://travis-ci.com/bloomberg/goldpinger.svg?branch=master)](https://travis-ci.com/bloomberg/goldpinger)

__Goldpinger__ makes calls between its instances for visibility and alerting.
It runs as a `DaemonSet` on `Kubernetes` and produces `Prometheus` metrics that can be scraped, visualised and alerted on.

Oh, and it gives you the graph below for your cluster. Check out the [video explainer](https://youtu.be/DSFxRz_0TU4).

![](./extras/screenshot.png)


## On the menu

- [Rationale](#rationale)
- [Quick start](#quick-start)
- [Building](#building)
  - [Compiling using a multi-stage Dockerfile](#compiling-using-a-multi-stage-dockerfile)
  - [Compiling locally](#compiling-locally)
- [Installation](#installation)
  - [Authentication with Kubernetes API](#authentication-with-kubernetes-api)
  - [Example YAML](#example-yaml)
- [Usage](#usage)
  - [UI](#ui)
  - [API](#api)
  - [Prometheus](#prometheus)
  - [Grafana](#grafana)
  - [Alert Manager](#alert-manager)
- [Contributions](#contributions)
- [License](#license)

## Rationale

We built __Goldpinger__ to troubleshoot, visualise and alert on our networking layer while adopting `Kubernetes` at Bloomberg. It has since become the go-to tool to see connectivity and slowness issues.

It's small, simple and you'll wonder why you hadn't had it before.

If you'd like to know more, you can watch [our presentation at Kubecon 2018 Seattle](https://youtu.be/DSFxRz_0TU4).


## Quick start

Getting from sources:

```sh
go get github.com/bloomberg/goldpinger/cmd/goldpinger
goldpinger --help
```

Getting from [docker hub](https://hub.docker.com/r/bloomberg/goldpinger):

```sh
# get from docker hub
docker pull bloomberg/goldpinger
```

Note, that in order to guarantee correct versions of dependencies, the project [uses `dep`](./Makefile).


## Building

The repo comes with two ways of building a `docker` image: compiling locally, and compiling using a multi-stage `Dockerfile` image. :warning: Depending on your `docker` setup, you might need to prepend the commands below with `sudo`.

### Compiling using a multi-stage Dockerfile

You will need `docker` version 17.05+ installed to support multi-stage builds.

```sh
# step 1: launch the build
make build-multistage

# step 2: push the image somewhere
namespace="docker.io/myhandle/" make tag
namespace="docker.io/myhandle/" make push
```

This was contributed via [@michiel](https://github.com/michiel) - kudos!

### Compiling locally

In order to build `Goldpinger`, you are going to need `go` version 1.10+, `dep`, and `docker`.

Building from source code consists of compiling the binary and building a [Docker image](./build/Dockerfile-simple):

```sh
# step 0: check out the code into your $GOPATH
go get github.com/bloomberg/goldpinger/cmd/goldpinger
cd $GOPATH/src/github.com/bloomberg/goldpinger

# step 1: download the dependencies via dep ensure
make vendor

# step 2: compile the binary for the desired architecture
make bin/goldpinger
# at this stage you should be able to run the binary
./bin/goldpinger --help

# step 3: build the docker image containing the binary
make build

# step 4: push the image somewhere
namespace="docker.io/myhandle/" make tag
namespace="docker.io/myhandle/" make push
```

## Installation

`Goldpinger` works by asking `Kubernetes` for pods with particular labels (`app=goldpinger`). While you can deploy `Goldpinger` in a variety of ways, it works very nicely as a `DaemonSet` out of the box.

### Authentication with Kubernetes API

`Goldpinger` supports using a `kubeconfig` (specify with `--kubeconfig-path`) or service accounts.

### Example YAML


Here's an example of what you can do (using the in-cluster authentication to `Kubernetes` apiserver).

```yaml
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: goldpinger
spec:
  updateStrategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app: goldpinger
      version: "1.5.0"
  template:
    metadata:
      labels:
        app: goldpinger
        version: "1.5.0"
    spec:
      containers:
        - name: goldpinger
          env:
            - name: HOST
              value: "0.0.0.0"
            - name: PORT
              value: "80"
            # injecting real hostname will make for easier to understand graphs/metrics
            - name: HOSTNAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            # podIP is used to select a randomized subset of nodes to ping.
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
          image: "docker.io/bloomberg/goldpinger:1.5.0"
          ports:
            - containerPort: 80
              name: http
          readinessProbe:
            httpGet:
              path: /healthz
              port: 80
            initialDelaySeconds: 20
            periodSeconds: 5
          livenessProbe:
            httpGet:
              path: /healthz
              port: 80
            initialDelaySeconds: 20
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: goldpinger
  labels:
    app: goldpinger
spec:
  type: NodePort
  ports:
    - port: 80
      nodePort: 30080
      name: http
  selector:
    app: goldpinger
```

Note, that you will also need to add an RBAC rule to allow `Goldpinger` to list other pods. If you're just playing around, you can consider a view-all default rule:

```yaml
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: view
subjects:
  - kind: ServiceAccount
    name: default
    namespace: default
```

You can also see [an example of using `kubeconfig` in the `./extras`](./extras/example-with-kubeconfig.yaml).

## Usage

### UI

Once you have it running, you can hit any of the nodes (port 30080 in the example above) and see the UI.

![](./extras/screenshot-big.png)

You can click on various nodes to gray out the clutter and see more information.

### API

The API exposed is via a well-defined [`Swagger` spec](./swagger.yml).

The spec is used to generate both the server and the client of `Goldpinger`. If you make changes, you can re-generate them using [go-swagger](https://github.com/go-swagger/go-swagger) via [`make swagger`](./Makefile)

### Prometheus

Once running, `Goldpinger` exposes `Prometheus` metrics at `/metrics`. All the metrics are prefixed with `goldpinger_` for easy identification.

You can see the metrics by doing a `curl http://$POD_ID:80/metrics`.

These are probably the droids you are looking for:

```sh
goldpinger_peers_response_time_s_*
goldpinger_peers_response_time_s_*
goldpinger_nodes_health_total
goldpinger_stats_total
goldpinger_errors_total
```

### Grafana

You can find an example of a `Grafana` dashboard that shows what's going on in your cluster in [extras](./extras/goldpinger-dashboard.json). This should get you started, and once you're on the roll, why not :heart: contribute some kickass dashboards for others to use?

### Alert Manager

Once you've gotten your metrics into `Prometheus`, you have all you need to set useful alerts.

To get you started, here's a rule that will trigger an alert if there are any nodes reported as unhealthy by any instance of `Goldpinger`.

```yaml
alert: goldpinger_nodes_unhealthy
expr: sum(goldpinger_nodes_health_total{status="unhealthy"})
  BY (goldpinger_instance) > 0
for: 5m
annotations:
  description: |
    Goldpinger instance {{ $labels.goldpinger_instance }} has been reporting unhealthy nodes for at least 5 minutes.
  summary: Instance {{ $labels.instance }} down
```

Similarly, why not :heart: contribute some amazing alerts for others to use?

## Contributions

We :heart: contributions.

Have you had a good experience with `Goldpinger`? Why not share some love and contribute code, dashboards and alerts?

If you're thinking of making some code changes, please be aware that most of the code is auto-generated from the `Swagger` spec. The spec is used to generate both the server and the client of `Goldpinger`. If you make changes, you can re-generate them using [go-swagger](https://github.com/go-swagger/go-swagger) via [`make swagger`](./Makefile).

Before you create that PR, please make sure you read [CONTRIBUTING](./CONTRIBUTING.md) and [DCO](./DCO.md).

## License

Please read the [LICENSE](./LICENSE) file here.

For each version built by travis, there is also an additional version, appended with `-vendor`, which contains all source code of the dependencies used in `goldpinger`.
