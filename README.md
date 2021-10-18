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
   --license value, -l value              path to Apiiro license file for custom Apiiro web sink
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
curl -s https://raw.githubusercontent.com/apiiro/kubescout/main/install.sh | sudo bash
# or for a specific version:
curl -s https://raw.githubusercontent.com/apiiro/kubescout/main/install.sh | sudo bash -s 1.4
```

If that doesn't work, try:

```bash
curl -s https://raw.githubusercontent.com/apiiro/kubescout/main/install.sh -o install.sh
sudo bash install.sh
```

then run: `kubescout -h`

## Test and Build

```bash
# run tests:
make test
# run integration tests:
make integration
# build binaries and run whole ci flow
make
```

## Crontab Installation

Add kubescout activation to the crontab to inspect the environment's pods daily

```bash
(crontab -l ; echo "0 0 * * * <path to kubescout binary> -n <name> -l <path to license file>") | crontab -
```

i.e

```bash
(crontab -l ; echo "0 0 * * * <path to kubescout binary> -n staging -l /opt/lim/apiiro.license") | crontab -
```

Make sure it is done with the right permissions. If `kubectl` requires sudo then apply `sudo -s` before running the
above command.

![meme](https://i.imgur.com/9nRSxD0.png)
