# Kube-Scout

![icon](kubescout.png)

```
NAME:
   kubescout - 0.1.0 - Scout for alarming issues in your Kubernetes cluster

USAGE:
   kubescout           --name value  [optional flags]

OPTIONS:
   --logs-tail value                      Length of logs tail when reporting of a problematic pod's logs (default: 250)
   --events-limit value                   Limits of namespace events to fetch (default: 150)
   --kubeconfig value, -c value           path to kubeconfig file, defaults to ~/.kube/config
   --time-format value, -f value          format for printing timestamps (default: "02 Jan 06 15:04 MST")
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

## Examples

```bash
kubescout --kubeconfig /root/.kube/config --name staging-cluster
kubescout --exclude-ns kube-system
kubescout --include-ns default,test,prod
```

## Install

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

## Roadmap

* Dockerfile, kube manifests, helm, job, cronjob
* Node resource alerts
* Node native service problems (kubelet, docker, containerd)
* Automatic fix actions
* Node defragmentation

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

## Crontab Setup

Add kubescout activation to the crontab to inspect the environment's pods on a schedule

```bash
(crontab -l ; echo "0 0 * * * <path to kubescout binary> -n <name>") | crontab -
```

Make sure it is done with the right permissions. If `kubectl` requires sudo then apply `sudo -s` before running the
above command.

![meme](https://i.imgur.com/9nRSxD0.png)
