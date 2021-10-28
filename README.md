# Kube-Scout

[![Open Source Love](https://badges.frapsoft.com/os/v2/open-source.svg?v=103)](https://github.com/ellerbrock/open-source-badges/)
[![CI](https://github.com/ReallyLiri/kubescout/actions/workflows/ci.yaml/badge.svg?)](https://github.com/ReallyLiri/kubescout/actions/workflows/ci.yaml)
[![Go project version](https://badge.fury.io/go/github.com%2Freallyliri%2Fkubescout.svg?)](https://badge.fury.io/go/github.com%2Freallyliri%2Fkubescout)
[![GoDoc](https://godoc.org/github.com/reallyliri/kubescout?status.svg)](https://pkg.go.dev/github.com/reallyliri/kubescout)

![icon](kubescout.png)

An alerting tool for Kubernetes clusters issues of all types, in real time, with intelligent redundancy, and easily extendable
api.

Output example:

```
Pod default/test-2-broken-image-7cbf974df9-gbnk9 is un-healthy:
Container test-2-broken-image still waiting due to ImagePullBackOff: Back-off pulling image "nginx:l4t3st"
Event by kubelet: Failed x4 since 27 Oct 21 14:20 UTC (last seen 2 minutes ago):
	Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown
Event by kubelet: Failed x4 since 27 Oct 21 14:20 UTC (last seen 2 minutes ago):
	Error: ErrImagePull
Event by kubelet: Failed x6 since 27 Oct 21 14:20 UTC (last seen 2 minutes ago):
	Error: ImagePullBackOff
----------------
Pod default/test-3-excessive-resources-699d58f55f-9gfft is un-healthy:
Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: 4 minutes ago)
Event by default-scheduler: FailedScheduling since 27 Oct 21 14:20 UTC (last seen 4 minutes ago):
	0/1 nodes are available: 1 Insufficient memory.
----------------
Pod default/test-4-crashlooping-dbdd84589-jvplc is un-healthy:
Container test-4-crashlooping is in CrashLoopBackOff: restarted 5 times, last exit due to Error (exit code 1)
Event by kubelet: BackOff x7 since 27 Oct 21 14:20 UTC (last seen 2 minutes ago):
	Back-off restarting failed container
Logs of container test-4-crashlooping:
--------
1
2
3
4
5
--------
----------------
Pod default/test-5-completed-757685986-r4tg2 is un-healthy:
Container test-5-completed is in CrashLoopBackOff: restarted 5 times, last exit due to Completed (exit code 0)
Event by kubelet: BackOff x8 since 27 Oct 21 14:20 UTC (last seen 2 minutes ago):
	Back-off restarting failed container
Logs of container test-5-completed:
--------
1
2
3
4
5
--------
----------------
Pod default/test-6-crashlooping-init-644545f5b7-bsvrn is un-healthy:
Container test-6-crashlooping-init-container (init) is in CrashLoopBackOff: restarted 5 times, last exit due to Error (exit code 1)
test-6-crashlooping-init-container (init) terminated due to Error (exit code 1)
Container test-6-crashlooping-init-container (init) restarted 5 times
Event by kubelet: BackOff x8 since 27 Oct 21 14:20 UTC (last seen 2 minutes ago):
	Back-off restarting failed container
Logs of container test-6-crashlooping-init-container:
--------
1
2
3
4
5
--------
----------------
```

Or in json format:

```json
{
  "alerts_by_cluster_name": {
    "minikube": [
      {
        "cluster_name": "minikube",
        "namespace": "default",
        "name": "test-2-broken-image-7cbf974df9-gbnk9",
        "kind": "Pod",
        "messages": [
          "Container test-2-broken-image still waiting due to ErrImagePull: rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown",
          "Container test-2-broken-image still waiting due to ImagePullBackOff: Back-off pulling image \"nginx:l4t3st\""
        ],
        "events": [
          "Event by kubelet: Failed x4 since 27 Oct 21 14:20 UTC (last seen 1 minute ago):\n\tFailed to pull image \"nginx:l4t3st\": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown",
          "Event by kubelet: Failed x4 since 27 Oct 21 14:20 UTC (last seen 1 minute ago):\n\tError: ErrImagePull",
          "Event by kubelet: Failed x6 since 27 Oct 21 14:20 UTC (last seen 1 minute ago):\n\tError: ImagePullBackOff"
        ],
        "timestamp": "2021-10-27T14:24:21.181725Z"
      },
      {
        "cluster_name": "minikube",
        "namespace": "default",
        "name": "test-3-excessive-resources-699d58f55f-9gfft",
        "kind": "Pod",
        "messages": [
          "Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: 3 minutes ago)"
        ],
        "events": [
          "Event by default-scheduler: FailedScheduling since 27 Oct 21 14:20 UTC (last seen 3 minutes ago):\n\t0/1 nodes are available: 1 Insufficient memory."
        ],
        "timestamp": "2021-10-27T14:24:21.181725Z"
      },
      {
        "cluster_name": "minikube",
        "namespace": "default",
        "name": "test-4-crashlooping-dbdd84589-jvplc",
        "kind": "Pod",
        "messages": [
          "Container test-4-crashlooping is in CrashLoopBackOff: restarted 5 times, last exit due to Error (exit code 1)"
        ],
        "events": [
          "Event by kubelet: BackOff x7 since 27 Oct 21 14:20 UTC (last seen 2 minutes ago):\n\tBack-off restarting failed container"
        ],
        "logs_by_container_name": {
          "test-4-crashlooping": "1\n2\n3\n4\n5"
        },
        "timestamp": "2021-10-27T14:24:21.181725Z"
      },
      {
        "cluster_name": "minikube",
        "namespace": "default",
        "name": "test-5-completed-757685986-r4tg2",
        "kind": "Pod",
        "messages": [
          "Container test-5-completed is in CrashLoopBackOff: restarted 5 times, last exit due to Completed (exit code 0)"
        ],
        "events": [
          "Event by kubelet: BackOff x8 since 27 Oct 21 14:20 UTC (last seen 2 minutes ago):\n\tBack-off restarting failed container"
        ],
        "logs_by_container_name": {
          "test-5-completed": "1\n2\n3\n4\n5"
        },
        "timestamp": "2021-10-27T14:24:21.181725Z"
      },
      {
        "cluster_name": "minikube",
        "namespace": "default",
        "name": "test-6-crashlooping-init-644545f5b7-bsvrn",
        "kind": "Pod",
        "messages": [
          "Container test-6-crashlooping-init-container (init) is in CrashLoopBackOff: restarted 5 times, last exit due to Error (exit code 1)"
        ],
        "events": [
          "Event by kubelet: BackOff x8 since 27 Oct 21 14:20 UTC (last seen 2 minutes ago):\n\tBack-off restarting failed container"
        ],
        "logs_by_container_name": {
          "test-6-crashlooping-init-container": "1\n2\n3\n4\n5"
        },
        "timestamp": "2021-10-27T14:24:21.181725Z"
      }
    ]
  }
}
```

![slack](https://i.imgur.com/03yuM55.png)

## Usage

### Kubernetes Native

TBD

### CLI

```
NAME:
   kubescout - 0.1.7 - Scout for alarming issues in your Kubernetes cluster

USAGE:
   kubescout                   [optional flags]

OPTIONS:
   --verbose, --vv                        Verbose logging (default: false)
   --logs-tail value                      Specifies the logs tail length when reporting logs from a problematic pod (default: 250)
   --events-limit value                   Maximum number of namespace events to fetch (default: 150)
   --kubeconfig value, -k value           kubeconfig file path, defaults to env var KUBECONFIG or ~/.kube/config
   --time-format value, -f value          timestamp print format (default: "02 Jan 06 15:04 MST")
   --locale value, -l value               timestamp print localization (default: "UTC")
   --pod-creation-grace-sec value         grace period in seconds since pod creation (default: 30)
   --pod-termination-grace-sec value      grace period in seconds since pod termination (default: 30)
   --pod-restart-grace-count value        grace count for pod restarts (default: 3)
   --node-resource-usage-threshold value  node resources usage threshold (default: 0.85)
   --exclude-ns value, -e value           namespaces to skip
   --include-ns value, -i value           namespaces to include (will skip any not listed if this option is used)
   --dedup-minutes value, -d value        time in minutes to silence duplicate or already observed alerts, or 0 to disable deduplication (default: 60)
   --store-filepath value, -s value       path to store file where state will be persisted or empty string to disable persistency (default: "kube-scout.store.json")
   --output value, -o value               output mode, one of pretty/json/yaml/discard (default: "pretty")
   --context value, -c value              context name to use from kubeconfig, defaults to current context
   --all-contexts, -a                     iterate all kubeconfig contexts, 'context' flag will be ignored if this flag is set (default: false)
   --exclude-contexts value               a comma separated list of kubeconfig context names to skip, only relevant if 'all-contexts' flag is set
   --help, -h                             show help (default: false)
   --version, -v                          print the version (default: false)
```

For example:

```bash
kubescout --kubeconfig /root/.kube/config --name staging-cluster
kubescout --exclude-ns kube-system
kubescout --include-ns default,test,prod
```

#### Install

```bash
curl -s https://raw.githubusercontent.com/reallyliri/kubescout/main/install.sh | sudo bash
# or for a specific version:
curl -s https://raw.githubusercontent.com/reallyliri/kubescout/main/install.sh | sudo bash -s 0.1.0
```

If that doesn't work, try:

```bash
curl -s https://raw.githubusercontent.com/reallyliri/kubescout/main/install.sh -o install.sh
sudo bash install.sh
```

then run: `kubescout -h`

### Package

You can also use the tool as a package from your code.

```
go get github.com/reallyliri/kubescout
```

```go
package example

import (
	"crypto/tls"
	"fmt"
	kubescoutconfig "github.com/reallyliri/kubescout/config"
	kubescout "github.com/reallyliri/kubescout/pkg"
	kubescoutsink "github.com/reallyliri/kubescout/sink"
	"net/http"
)

func main() {

	// simple default execution:
	_ = kubescout.Scout(nil, nil)

	// example using Slack webhook as sink:
	cfg, _ := kubescoutconfig.DefaultConfig()
	cfg.KubeconfigFilePath = "/root/configs/staging-kubeconfig"
	sink, _ := kubescoutsink.CreateWebSink(
		"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
		func() (http.RoundTripper, error) {
			skipVerifyTransport := http.DefaultTransport.(*http.Transport).Clone()
			skipVerifyTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			return skipVerifyTransport, nil
		},
		func(request *http.Request) error {
			request.Header.Add("Content-Type", "application/json")
			return nil
		},
		func(response *http.Response, responseBody string) error {
			if responseBody != "ok" {
				return fmt.Errorf("non-ok response from Slack: '%v'", responseBody)
			}
			return nil
		},
		false,
	)
	_ = kubescout.Scout(cfg, sink)
}
```

## Test and Build

```bash
# vet and lint
go vet
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:latest-alpine golangci-lint run --deadline=180s
# tests
go test -v ./...
# integration tests (requires minikube)
go test -v --tags=integration ./integration_test.go
# build
CGO_ENABLED=0 go build -o bin/kubescout-$(go run . --version | cut -d" " -f 3) .
```

![meme](https://i.imgur.com/9nRSxD0.png)
