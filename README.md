# Brigade Kubernetes Gateway

**Experimental:** This should not be used in production. Misconfiguration can
consume massive amounts of cluster resources.

This is a Brigade gateway that listens to the Kubernetes event stream and triggers
events inside of Brigade.

Issues for Brigade projects are all tracked [on the main Brigade project](https://github.com/Azure/brigade/issues).

## Installation

To install the latest image into your cluster:

```
$ helm inspect values charts/brigade-k8s-gateway > myvalues.yaml
# edit myvalues.yaml
$ helm install -f myvalues charts/brigade-k8s-gateway
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


## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
