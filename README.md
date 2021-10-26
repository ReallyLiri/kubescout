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
Found 5 alerts for cluster minikube:
Pod default/test-2-broken-image-7cbf974df9-6zvr8 is un-healthy:
Pod is in Pending phase
test-2-broken-image still waiting due to ErrImagePull: rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown
Event by kubelet: Failed x4 since 25 Oct 21 07:26 UTC (last seen 15 seconds ago):
	Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown
Event by kubelet: Failed x4 since 25 Oct 21 07:26 UTC (last seen 15 seconds ago):
	Error: ErrImagePull
Event by kubelet: Failed x5 since 25 Oct 21 07:26 UTC (last seen 1 second ago):
	Error: ImagePullBackOff
----------------
Pod default/test-3-excessive-resources-699d58f55f-rxr2l is un-healthy:
Pod is in Pending phase
Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: 1 minute ago)
Event by default-scheduler: FailedScheduling since 25 Oct 21 07:26 UTC (last seen 1 minute ago):
	0/1 nodes are available: 1 Insufficient memory.
----------------
Pod default/test-4-crashlooping-dbdd84589-j6r8v is un-healthy:
test-4-crashlooping terminated due to Error (exit code 1)
test-4-crashlooping had restarted 4 times, last exit due to Error (exit code 1)
Event by kubelet: BackOff x8 since 25 Oct 21 07:26 UTC (last seen 19 seconds ago):
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
Pod default/test-5-completed-757685986-s58rv is un-healthy:
test-5-completed still waiting due to CrashLoopBackOff: back-off 1m20s restarting failed container
test-5-completed had restarted 4 times, last exit due to Completed (exit code 0)
Event by kubelet: BackOff x7 since 25 Oct 21 07:26 UTC (last seen 25 seconds ago):
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
Pod default/test-6-crashlooping-init-644545f5b7-sffvr is un-healthy:
Pod is in Pending phase
test-6-crashlooping-init-container (init) terminated due to Error (exit code 1)
test-6-crashlooping-init-container (init) had restarted 4 times, last exit due to Error (exit code 1)
Event by kubelet: BackOff x8 since 25 Oct 21 07:26 UTC (last seen 20 seconds ago):
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
        "name": "test-2-broken-image-7cbf974df9-6zvr8",
        "kind": "Pod",
        "messages": [
          "Pod is in Pending phase",
          "test-2-broken-image still waiting due to ImagePullBackOff: Back-off pulling image \"nginx:l4t3st\""
        ],
        "events": [
          "Event by kubelet: Failed x4 since 25 Oct 21 07:26 UTC (last seen 49 seconds ago):\n\tFailed to pull image \"nginx:l4t3st\": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown",
          "Event by kubelet: Failed x4 since 25 Oct 21 07:26 UTC (last seen 49 seconds ago):\n\tError: ErrImagePull",
          "Event by kubelet: Failed x6 since 25 Oct 21 07:26 UTC (last seen 23 seconds ago):\n\tError: ImagePullBackOff"
        ],
        "logs_by_container_name": {},
        "timestamp": "2021-10-25T07:28:49.24639Z"
      },
      {
        "cluster_name": "minikube",
        "namespace": "default",
        "name": "test-3-excessive-resources-699d58f55f-rxr2l",
        "kind": "Pod",
        "messages": [
          "Pod is in Pending phase",
          "Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: 2 minutes ago)"
        ],
        "events": [
          "Event by default-scheduler: FailedScheduling since 25 Oct 21 07:26 UTC (last seen 2 minutes ago):\n\t0/1 nodes are available: 1 Insufficient memory."
        ],
        "logs_by_container_name": {},
        "timestamp": "2021-10-25T07:28:49.24639Z"
      },
      {
        "cluster_name": "minikube",
        "namespace": "default",
        "name": "test-4-crashlooping-dbdd84589-j6r8v",
        "kind": "Pod",
        "messages": [
          "test-4-crashlooping still waiting due to CrashLoopBackOff: back-off 1m20s restarting failed container",
          "test-4-crashlooping had restarted 4 times, last exit due to Error (exit code 1)"
        ],
        "events": [
          "Event by kubelet: BackOff x8 since 25 Oct 21 07:26 UTC (last seen 53 seconds ago):\n\tBack-off restarting failed container"
        ],
        "logs_by_container_name": {
          "test-4-crashlooping": "1\n2\n3\n4\n5"
        },
        "timestamp": "2021-10-25T07:28:49.24639Z"
      },
      {
        "cluster_name": "minikube",
        "namespace": "default",
        "name": "test-5-completed-757685986-s58rv",
        "kind": "Pod",
        "messages": [
          "test-5-completed still waiting due to CrashLoopBackOff: back-off 1m20s restarting failed container",
          "test-5-completed had restarted 4 times, last exit due to Completed (exit code 0)"
        ],
        "events": [
          "Event by kubelet: BackOff x7 since 25 Oct 21 07:26 UTC (last seen 59 seconds ago):\n\tBack-off restarting failed container"
        ],
        "logs_by_container_name": {
          "test-5-completed": "1\n2\n3\n4\n5"
        },
        "timestamp": "2021-10-25T07:28:49.24639Z"
      },
      {
        "cluster_name": "minikube",
        "namespace": "default",
        "name": "test-6-crashlooping-init-644545f5b7-sffvr",
        "kind": "Pod",
        "messages": [
          "Pod is in Pending phase",
          "test-6-crashlooping-init-container (init) still waiting due to CrashLoopBackOff: back-off 1m20s restarting failed container",
          "test-6-crashlooping-init-container (init) had restarted 4 times, last exit due to Error (exit code 1)"
        ],
        "events": [
          "Event by kubelet: BackOff x8 since 25 Oct 21 07:26 UTC (last seen 54 seconds ago):\n\tBack-off restarting failed container"
        ],
        "logs_by_container_name": {
          "test-6-crashlooping-init-container": "1\n2\n3\n4\n5"
        },
        "timestamp": "2021-10-25T07:28:49.24639Z"
      }
    ]
  }
}
```

## Usage

### Kubernetes Native

TBD

### CLI

```
NAME:
   kubescout - 0.1.5 - Scout for alarming issues in your Kubernetes cluster

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
   --iterations value, --it value         number of diag iterations, meant to better capture constantly changing states (default: 3)
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
	configuration, _ := kubescoutconfig.DefaultConfig()
	configuration.KubeconfigFilePath = "/root/configs/staging-kubeconfig"
	sink, _ := kubescoutsink.CreateWebSink(
		"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
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
	_ = kubescout.Scout(configuration, sink)
}
```

## Test and Build

```bash
# vet and lint
go vet
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:latest-alpine golangci-lint run --deadline=65s
# tests
go test -v ./...
# integration tests (requires minikube)
go test -v --tags=integration ./integration_test.go
# build
CGO_ENABLED=0 go build -o bin/kubescout-$(go run . --version | cut -d" " -f 3) .
```

![meme](https://i.imgur.com/9nRSxD0.png)
