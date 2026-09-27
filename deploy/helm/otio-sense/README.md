# One Sense collector with an existing MQTT broker

This chart deploys one Sense collector and one Protocol Lab gateway. It creates no database, broker, simulator, PVC or public ingress. It is not the HA chart.

In Protocol Lab, test your device connection, open **Deploy Sense**, enter the broker and image names, and download `sense-values.json`. Build and push both images into a registry reachable by the cluster, or load them onto all local cluster nodes. Run from the repository root:

```sh
helm upgrade --install sense-edge deploy/helm/otio-sense -f sense-values.json
kubectl port-forward service/sense-edge 8100:8100
```

The generated values use release name `sense-edge`. For another release name, change every source's `options.gatewayURL` to `http://<release>-gateway:8500`. Device and broker addresses must be reachable from cluster pods. Configure `imagePullSecrets` for a private registry.

The configuration is stored in a Kubernetes Secret and mounted read-only. Change the values and run `helm upgrade` to apply edits. A configuration checksum triggers the rollout. The collector uses a Recreate strategy to avoid overlapping MQTT client IDs. Generated examples assume an anonymous broker on port 1883; authenticated or TLS brokers require adapting the deployment to the supported MQTT environment settings. This minimal chart does not expose those environment settings as values. Give each collector a unique broker client ID and each device a distinct agent ID before deploying multiple instances.

Read adapters use a separate gateway service; it has no active simulator by default. This separation preserves the existing Sense adapter contract. These examples are not a lossless, exactly-once or high-availability deployment; use [Example 07](../../../examples/07-kubernetes-ha/README.md) for HA.
