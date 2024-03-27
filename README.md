# kubectl plugin to forcefully delete a kubernetes namespace
[![release](https://github.com/doodlescheduling/kubectl-del-namespace/actions/workflows/release.yaml/badge.svg)](https://github.com/doodlescheduling/kubectl-del-namespace/actions/workflows/release.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/doodlescheduling/kubectl-del-namespace)](https://goreportcard.com/report/github.com/doodlescheduling/kubectl-del-namespace)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/kubectl-del-namespace/badge)](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/kubectl-del-namespace)
[![Coverage Status](https://coveralls.io/repos/github/DoodleScheduling/kubectl-del-namespace/badge.svg?branch=master)](https://coveralls.io/github/DoodleScheduling/kubectl-del-namespace?branch=master)

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
docker pull ghcr.io/doodlescheduling/kubectl-del-namespace:[version]
```

## Usage

```
kubectl del-namespace [namespace]
```

This runs in a dry-mode and eventually also prints all resources with finalizers.
Invoke it with the `--yes` flag will delete the namespace and will make sure the namespace gets deleted.
```
kubectl del-namespace [namespace] --yes
``` 

**Note**: It is recommeneded to specify the resources which are allowed to be cleaned up by ignoring any finalizers and optionally 
setting an appropriate grace-period to allow controllers to eventually clean up.
```
kubectl del-namespace [namespace] --yes --grace-period=30s --resources helmreleases.helm.toolkit.fluxcd.io,keycloakinfinispanclusters.keycloak.infra.doodle.com,growthbookinstances.growthbook.infra.doodle.com
```

## Help
```
Force delete a kubernetes namespace including all resources with blocking finalizers

Usage:
  kubectl del-namespace [namespace] [flags]

Flags:
      --add_dir_header                   If true, adds the file directory to the header of the log messages
      --alsologtostderr                  log to standard error as well as files (no effect when -logtostderr=true)
      --as string                        Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
      --as-group stringArray             Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                    UID to impersonate for the operation.
      --cache-dir string                 Default cache directory (default "/home/raffael/.kube/cache")
      --certificate-authority string     Path to a cert file for the certificate authority
      --client-certificate string        Path to a client certificate file for TLS
      --client-key string                Path to a client key file for TLS
      --cluster string                   The name of the kubeconfig cluster to use
      --context string                   The name of the kubeconfig context to use
      --disable-compression              If true, opt-out of response compression for all requests to the server
      --grace-period duration            Force remove all finalizers after the grace period was reached (default 10s)
  -h, --help                             help for kubectl
      --insecure-skip-tls-verify         If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string                Path to the kubeconfig file to use for CLI requests.
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory (no effect when -logtostderr=true)
      --log_file string                  If non-empty, use this log file (no effect when -logtostderr=true)
      --log_file_max_size uint           Defines the maximum size a log file can grow to (no effect when -logtostderr=true). Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                      log to standard error instead of files (default true)
  -n, --namespace string                 If present, the namespace scope for this CLI request
      --one_output                       If true, only write logs to their native severity level (vs also writing to each lower severity level; no effect when -logtostderr=true)
      --request-timeout string           The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
      --resources strings                Whitelist resources from which the finalizers are removed. If not set all resources are targeted.
  -s, --server string                    The address and port of the Kubernetes API server
      --skip_headers                     If true, avoid header prefixes in the log messages
      --skip_log_headers                 If true, avoid headers when opening log files (no effect when -logtostderr=true)
      --stderrthreshold severity         logs at or above this threshold go to stderr when writing to files and stderr (no effect when -logtostderr=true or -alsologtostderr=true) (default 2)
      --tls-server-name string           Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                     Bearer token for authentication to the API server
      --user string                      The name of the kubeconfig user to use
  -v, --v Level                          number for the log level verbosity
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
      --yes                              Force remove all finalizers
```
