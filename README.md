> ⚠️&nbsp;&nbsp;This repo contains the source for a component of the Brigade
> v1.x ecosystem. Brigade v1.x reached end-of-life on June 1, 2022 and as a
> result, this component is no longer maintained.

# Brigade Kubernetes Gateway

**Experimental:** This should not be used in production. Misconfiguration can
consume massive amounts of cluster resources.

This is a Brigade gateway that listens to the Kubernetes event stream and triggers
events inside of Brigade.

Issues for Brigade projects are all tracked [on the main Brigade project](https://github.com/brigadecore/brigade/issues).

## Installation

The [Brigade K8s Gateway Helm Chart][brigade-k8s-gateway-chart] is hosted at the
[brigadecore/charts][charts] repository.

To install the latest image into your cluster:

```
$ helm repo add brigade https://brigadecore.github.io/charts
$ helm inspect values brigade/brigade-k8s-gateway > myvalues.yaml
# edit myvalues.yaml
$ helm install -f myvalues brigade/brigade-k8s-gateway
```

### Building from Source

You must have the Go toolchain, make, and dep installed. For Docker support, you
will need to have Docker installed as well. From there:

```
$ make build
```

To build a Docker image, you can `make docker-build`.

## Configuring

Configuring the gateway is tricky: You don't want to cause a build to trigger
another build. In your Helm `values.yaml` file you will want to configure your
filters appropriately.

Here is an example that listens to Pod events that occur in the namespace
`pequod`.

```yaml
filters:
  # Ignore all events coming from kube-system
  - namespace: kube-system
    action: reject
  # Ignore events on Nodes. We just care about Pods
  - kind: Node
    action: reject
  # Ignore "Killing" messages for Pods
  - kind: Pod
    reasons:
      - Killing
    action: reject
  # ONLY Listen to events for Pods in this namespace
  - kind: Pod
    namespace: pequod
    action: accept
  # Reject anything else (don't DOS yourself)
  - action: reject
```

For example, the following kinds (and more) produce events

- Node
- Pod
- CronJob
- Job
- Deployment
- ReplicaSet

The list of reasons is unconstrained (the value is a string in the Kubernetes
API). But here are a few examples


- Node `Starting`: A node is starting up
- Pod `Killing`: Triggered when a pod has been terminated
- ReplicaSet `SuccessfulCreate`: Triggered when a ReplicaSet has been created

To make it easier to see what the gateway sees, we log the events. You can use
`kubectl logs $GATEWAY_POD_NAME` to see the data. HEre's an example log entry
for a `Pod`'s `Pulled` event:

```
Processing default/wp-wordpress-69cfcc7544-nmsj5.1510e390827104e3: {
  "metadata": {
    "name": "wp-wordpress-69cfcc7544-nmsj5.1510e390827104e3",
    "namespace": "default",
    "selfLink": "/api/v1/namespaces/default/events/wp-wordpress-69cfcc7544-nmsj5.1510e390827104e3",
    "uid": "c2e459a8-0b9d-11e8-850f-080027ff61a5",
    "resourceVersion": "95112",
    "creationTimestamp": "2018-02-07T00:28:04Z"
  },
  "involvedObject": {
    "kind": "Pod",
    "namespace": "default",
    "name": "wp-wordpress-69cfcc7544-nmsj5",
    "uid": "ac2aa534-0b9d-11e8-850f-080027ff61a5",
    "apiVersion": "v1",
    "resourceVersion": "95051",
    "fieldPath": "spec.containers{wp-wordpress}"
  },
  "reason": "Pulled",
  "message": "Container image \"bitnami/wordpress:4.9.1-r0\" already present on machine",
  "source": {
    "component": "kubelet",
    "host": "minikube"
  },
  "firstTimestamp": "2018-02-07T00:28:04Z",
  "lastTimestamp": "2018-02-07T00:28:04Z",
  "count": 1,
  "type": "Normal"
}
```

### RBAC

If you are running with RBAC, you will need to write roles and role bindings for
the namespaces you want this service to attach to. The chart includes a role/role
binding for the `default` namespace. You may use this as a template.


# Contributing

This Brigade project accepts contributions via GitHub pull requests. This document outlines the process to help get your contribution accepted.

## Signed commits

A DCO sign-off is required for contributions to repos in the brigadecore org.  See the documentation in
[Brigade's Contributing guide](https://github.com/brigadecore/brigade/blob/master/CONTRIBUTING.md#signed-commits)
for how this is done.

[charts]: https://github.com/brigadecore/charts
[brigade-k8s-gateway-chart]: https://github.com/brigadecore/charts/tree/master/charts/brigade-k8s-gateway