# Kube-Scout

[![Open Source Love](https://badges.frapsoft.com/os/v2/open-source.svg?v=103)](https://github.com/ellerbrock/open-source-badges/)
[![CI](https://github.com/ReallyLiri/kubescout/actions/workflows/ci.yaml/badge.svg?)](https://github.com/ReallyLiri/kubescout/actions/workflows/ci.yaml)
[![Go project version](https://badge.fury.io/go/github.com%2Freallyliri%2Fkubescout.svg?)](https://badge.fury.io/go/github.com%2Freallyliri%2Fkubescout)
[![GoDoc](https://godoc.org/github.com/reallyliri/kubescout?status.svg)](https://pkg.go.dev/github.com/reallyliri/kubescout)

![icon](kubescout.png)

Tool to alert on Kubernetes cluster issues of all kinds, in real time, with smart redundancy and with simple extendable
api.

Output example:

```
Found 13 alerts for cluster minikube:
Pod default/test-2-broken-image-7cbf974df9-g2tqp is un-healthy
        Pod is in Pending phase
        test-2-broken-image still waiting due to ImagePullBackOff: Back-off pulling image "nginx:l4t3st"
----------------
Event on Pod test-2-broken-image-7cbf974df9-g2tqp due to Failed (at 21 Oct 21 06:21 UTC, 7 minutes ago):
        Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown
----------------
Event on Pod test-2-broken-image-7cbf974df9-g2tqp due to Failed (at 21 Oct 21 06:21 UTC, 7 minutes ago):
        Error: ErrImagePull
----------------
Event on Pod test-2-broken-image-7cbf974df9-g2tqp due to Failed (at 21 Oct 21 06:22 UTC, 7 minutes ago):
        Error: ImagePullBackOff
----------------
Pod default/test-3-excessive-resources-699d58f55f-52xbb is un-healthy
        Pod is in Pending phase
        Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: 9 minutes ago)
----------------
Event on Pod test-3-excessive-resources-699d58f55f-52xbb due to FailedScheduling (at unavailable time, unknown time ago):
        0/1 nodes are available: 1 Insufficient memory.
----------------
Pod default/test-4-crashlooping-dbdd84589-mwtxk is un-healthy
        test-4-crashlooping still waiting due to CrashLoopBackOff: back-off 5m0s restarting failed container
        test-4-crashlooping had restarted 6 times, last exit due to Error (exit code 1)
logs of container test-4-crashlooping:
<<<<<<<<<<
...
>>>>>>>>>>
----------------
Event on Pod test-4-crashlooping-dbdd84589-mwtxk due to BackOff (at 21 Oct 21 06:25 UTC, 4 minutes ago):
        Back-off restarting failed container
----------------
Pod default/test-5-completed-757685986-7v2hm is un-healthy
        test-5-completed still waiting due to CrashLoopBackOff: back-off 5m0s restarting failed container
        test-5-completed had restarted 6 times, last exit due to Completed (exit code 0)
logs of container test-5-completed:
<<<<<<<<<<
...
>>>>>>>>>>
----------------
Event on Pod test-5-completed-757685986-7v2hm due to BackOff (at 21 Oct 21 06:24 UTC, 4 minutes ago):
        Back-off restarting failed container
----------------
Pod default/test-6-crashlooping-init-644545f5b7-xpfsb is un-healthy
        Pod is in Pending phase
        test-6-crashlooping-init-container (init) still waiting due to CrashLoopBackOff: back-off 5m0s restarting failed container
        test-6-crashlooping-init-container (init) had restarted 6 times, last exit due to Error (exit code 1)
logs of container test-6-crashlooping-init-container:
<<<<<<<<<<
...
>>>>>>>>>>
----------------
Event on Pod test-6-crashlooping-init-644545f5b7-xpfsb due to BackOff (at 21 Oct 21 06:25 UTC, 4 minutes ago):
        Back-off restarting failed container
```

Or in json format:

```json
{
  "cluster_name": "minikube",
  "content": [
    "Pod default/test-2-broken-image-7cbf974df9-g2tqp is un-healthy\n\tPod is in Pending phase\n\ttest-2-broken-image still waiting due to ImagePullBackOff: Back-off pulling image \"nginx:l4t3st\"",
    "Event on Pod test-2-broken-image-7cbf974df9-g2tqp due to Failed (at 21 Oct 21 06:20 UTC, 10 seconds ago):\n\tFailed to pull image \"nginx:l4t3st\": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown",
    "Event on Pod test-2-broken-image-7cbf974df9-g2tqp due to Failed (at 21 Oct 21 06:20 UTC, 10 seconds ago):\n\tError: ErrImagePull",
    "Event on Pod test-2-broken-image-7cbf974df9-g2tqp due to Failed (at 21 Oct 21 06:20 UTC, 25 seconds ago):\n\tError: ImagePullBackOff",
    "Pod default/test-3-excessive-resources-699d58f55f-52xbb is un-healthy\n\tPod is in Pending phase\n\tUnschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: 42 seconds ago)",
    "Event on Pod test-3-excessive-resources-699d58f55f-52xbb due to FailedScheduling (at unavailable time, unknown time ago):\n\t0/1 nodes are available: 1 Insufficient memory.",
    "Pod default/test-4-crashlooping-dbdd84589-mwtxk is un-healthy\n\ttest-4-crashlooping terminated due to Error (exit code 1)\nlogs of container test-4-crashlooping:\n\u003c\u003c\u003c\u003c\u003c\u003c\u003c\u003c\u003c\u003c\n1\n2\n3\n4\n5\n\n\u003e\u003e\u003e\u003e\u003e\u003e\u003e\u003e\u003e\u003e",
    "Event on Pod test-4-crashlooping-dbdd84589-mwtxk due to BackOff (at 21 Oct 21 06:20 UTC, 7 seconds ago):\n\tBack-off restarting failed container",
    "Pod default/test-5-completed-757685986-7v2hm is un-healthy\n\ttest-5-completed still waiting due to CrashLoopBackOff: back-off 10s restarting failed container\nlogs of container test-5-completed:\n\u003c\u003c\u003c\u003c\u003c\u003c\u003c\u003c\u003c\u003c\n1\n2\n3\n4\n5\n\n\u003e\u003e\u003e\u003e\u003e\u003e\u003e\u003e\u003e\u003e",
    "Event on Pod test-5-completed-757685986-7v2hm due to BackOff (at 21 Oct 21 06:20 UTC, 18 seconds ago):\n\tBack-off restarting failed container",
    "Pod default/test-6-crashlooping-init-644545f5b7-xpfsb is un-healthy\n\tPod is in Pending phase\n\ttest-6-crashlooping-init-container (init) terminated due to Error (exit code 1)\nlogs of container test-6-crashlooping-init-container:\n\u003c\u003c\u003c\u003c\u003c\u003c\u003c\u003c\u003c\u003c\n1\n2\n3\n4\n5\n\n\u003e\u003e\u003e\u003e\u003e\u003e\u003e\u003e\u003e\u003e",
    "Event on Pod test-6-crashlooping-init-644545f5b7-xpfsb due to BackOff (at 21 Oct 21 06:20 UTC, 3 seconds ago):\n\tBack-off restarting failed container"
  ]
}
```

## Roadmap

* Usage: Dockerfile, kube manifests, helm, job, cronjob
* Feature: Node resources (disk, inodes, processes)
* Feature: Node native service problems (kubelet, docker, containerd)
* Feature: Automatic fix actions
* Feature: Nodes defragmentation

## Usage

### Kubernetes Native

TBD

### CLI

```
NAME:
   kubescout - 0.1.3 - Scout for alarming issues in your Kubernetes cluster

USAGE:
   kubescout             --name value   [optional flags]

OPTIONS:
   --verbose, --vv                        Log verbose (default: false)
   --logs-tail value                      Length of logs tail when reporting of a problematic pod's logs (default: 250)
   --events-limit value                   Limits of namespace events to fetch (default: 150)
   --kubeconfig value, -c value           path to kubeconfig file, defaults to ~/.kube/config
   --time-format value, -f value          format for printing timestamps (default: "02 Jan 06 15:04 MST")
   --locale value, -l value               localization to use when printing timestamps (default: "UTC")
   --pod-creation-grace-sec value         grace time in seconds since pod creation (default: 30)
   --pod-termination-grace-sec value      grace time in seconds since pod termination (default: 30)
   --pod-restart-grace-count value        grace time in seconds since pod termination (default: 3)
   --node-resource-usage-threshold value  node resources usage threshold (default: 0.85)
   --exclude-ns value, -e value           namespaces to skip
   --include-ns value, -i value           namespaces to include (will skip any not listed if this option is used)
   --name value, -n value                 name of the scouted cluster
   --dedup-minutes value, -d value        number of minutes to silence duplicated or already observed alerts or 0 if this feature should not be applied (default: 60)
   --store-filepath value, -s value       path to store file where duplicated message information will be persisted or empty string if this feature should not be applied (default: "kube-scout.store.json")
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

The tool can also be used as a package from your code.

```
go get github.com/reallyliri/kubescout
```

```go
package example
import kubescout "github.com/reallyliri/kubescout/pkg"
import kubescoutconfig "github.com/reallyliri/kubescout/config"
import kubescoutsink "github.com/reallyliri/kubescout/sink"

func main1() {
	_ = kubescout.Scout(nil, nil)
}

func main2() {
	configuration, _ := kubescoutconfig.DefaultConfig()
	sink, _ := kubescoutsink.CreateSlackSink("https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX")
	_ = kubescout.Scout(configuration, sink)
}

func main3() {
	configuration, _ := kubescoutconfig.DefaultConfig()
	configuration.KubeconfigFilePath = "/root/configs/staging-kubeconfig"
	sink, _ := kubescoutsink.CreateWebSink("https://post.url", nil, false)
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
GO111MODULE=on CGO_ENABLED=0 $(GOCMD) build -o bin/$kubescout-$(shell $(GOCMD) run . --version | cut -d" " -f 3) .
```

![meme](https://i.imgur.com/9nRSxD0.png)
