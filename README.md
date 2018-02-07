# Brigade Kubernetes Gateway

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

## Example

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

When it comes to writing `brigade.js` scripts that support this gateway, the event
fired will be named according to the reason it was fired:

- `starting`: A pod or node is starting up
- `killing`: Triggered when a pod has been terminated
- ...

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
