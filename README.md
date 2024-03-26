# kubectl plugin to forcefully delete a kubernetes namespace
[![release](https://github.com/doodlescheduling/kubectl-del-namespace/actions/workflows/release.yaml/badge.svg)](https://github.com/doodlescheduling/kubectl-del-namespace/actions/workflows/release.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/doodlescheduling/kubectl-del-namespace)](https://goreportcard.com/report/github.com/doodlescheduling/kubectl-del-namespace)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/kubectl-del-namespace/badge)](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/kubectl-del-namespace)
[![Coverage Status](https://coveralls.io/repos/github/DoodleScheduling/kubectl-del-namespace/badge.svg?branch=master)](https://coveralls.io/github/DoodleScheduling/kubectl-del-namespace?branch=master)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kubectl-del-namespace)](https://artifacthub.io/packages/search?repo=kubectl-del-namespace)

This kubectl plugin will help to forcefully purge an entire kubernetes namespace.
Yes the kube-apiserver does this already when a namespace is deleted the usual way.
However resources with finalizers will block the deletion of the namespace until they are itself deleted.
And various resources can end in a dead lock because the namespace delete request deleted resource dependencies first 
and the resource reconciliation blocks their deletion because of that.

This plugin will enforce the removal of all the finalizers. This is fine if all these resources managed namespace scoped resources living within the namespace
which gets deleted anyways. However this operation can still lead to zombie resources outside the cluster if such resources are blocking the deletion and
you need to be aware of what these resources do!
The default dry-run will print all resources with finalizers.

## Installation

### Brew
```sh
brew tap doodlescheduling/kubectl-del-namespace
brew install kubectl-del-namespace
```

### Docker
```sh
docker pull ghcr.io/doodlescheduling/kubectl-del-namespace:latest
```

## Usage

```
kubectl del-namespace [namespace]
```

This runs in a dry-mode and eventually also prints all resources with finalizers.
Invoke it with the `--yes` flag will delete the namespace and will make sure the namespace gets deleted.
```
kubectl del-namespace [namespace] ---yes
``` 