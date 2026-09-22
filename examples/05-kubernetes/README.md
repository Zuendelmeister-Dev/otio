# Example 05 — Kubernetes Protocol Lab

Deploys Protocol Lab (including native Presense simulators), Sense, Lense, Mosquitto, RabbitMQ and Postgres into namespace `otio`. Uses Kustomize built into kubectl; Helm is not required. Dispense and the legacy standalone Presense demos are not included in this example.

## Requirements

- A Kubernetes cluster, kubectl with a selected context, and permission to create a namespace, Deployments, Services, ConfigMaps, Secrets and PVCs.
- A default StorageClass providing at least 3 GiB for the two PVCs. kind supplies a local provisioner. On another cluster, configure `storageClassName` in an overlay if there is no default.
- Docker for building the application images. Allow roughly 4 GiB RAM for the demo cluster plus additional memory for the Go image builds.

All commands below run from the repository root. Check the target before applying:

```sh
kubectl config current-context
```

## Local example with kind

Install [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) and start Docker, then:

```sh
kind create cluster --name otio
docker build -t otio/protocol-lab:local -f apps/protocol-lab/Dockerfile .
docker build -t otio/sense:local -f apps/sense/Dockerfile .
docker build -t otio/lense:local -f apps/lense/Dockerfile .
kind load docker-image --name otio otio/protocol-lab:local otio/sense:local otio/lense:local
kubectl kustomize examples/05-kubernetes
kubectl apply -k examples/05-kubernetes
kubectl -n otio wait --for=condition=Available deployment --all --timeout=300s
kubectl -n otio get pods,svc,pvc
```

The first build downloads the Go toolchain and protocol dependencies. `IfNotPresent` allows the kind nodes to use the loaded local images. After rebuilding the same tags, load them again and run `kubectl -n otio rollout restart deployment/protocol-lab deployment/sense deployment/lense`.

## Open the UIs

Run each port-forward in its own terminal and keep it running:

```sh
kubectl -n otio port-forward service/lense 8000:8000
kubectl -n otio port-forward service/sense 8100:8100
kubectl -n otio port-forward service/protocol-lab 8500:8500
```

Open Lense at http://localhost:8000, Sense at http://localhost:8100 and Protocol Lab at http://localhost:8500/static/. The Protocol Lab link inside Lense also works through its proxy using only the Lense port-forward. Direct component links need the corresponding port-forward. No Ingress controller, LoadBalancer or publicly exposed port is required.

Change the simulator temperature, read Modbus TCP (`holding-register:1:UINT`, raw temperature ×100), RTU-over-TCP or OPC UA (`ns=1;s=Temperature`), and check the five sources in Lense. For broker reads use `tcp://mqtt:1883` and `amqp://lab:lab@rabbitmq:5672/`, with addresses `lab/temperature` and `amq.topic/lab.temperature`. Requests originate inside Protocol Lab; `localhost` refers to that Pod, not your computer.

To test a native endpoint from your computer, forward its port too, for example `kubectl -n otio port-forward service/protocol-lab 4842:4842`. To reach real TCP/IP devices, the cluster network must route to those devices. No fieldbus hardware or non-IP protocol stack is deployed.

## Existing cluster / registry images

Build and push the three application images under your own registry and immutable version tag. Add an `images` section to a copy of this example's Kustomization, for example:

```yaml
images:
  - name: otio/protocol-lab
    newName: registry.example.com/otio/protocol-lab
    newTag: release-1
  - name: otio/sense
    newName: registry.example.com/otio/sense
    newTag: release-1
  - name: otio/lense
    newName: registry.example.com/otio/lense
    newTag: release-1
```

Configure `imagePullSecrets` on the Pods for a private registry. Build for the cluster's CPU architecture. Replace the example's public demo credentials with a managed Secret containing the same keys; the reusable base expects a Secret named `otio-credentials`. The AMQP URL must use the same RabbitMQ credentials. Also update the AMQP connection in Sense's seed before first deployment, or through the configuration UI for an existing deployment. Changing a Kubernetes Secret does not rotate credentials in an already initialized Postgres database. This remains a beta lab: anonymous MQTT, unencrypted demo traffic and unauthenticated UIs require a trusted namespace/network.

## Configuration, persistence and rollout

The reusable manifests are in [`deploy/kubernetes`](../../deploy/kubernetes). ConfigMap name hashes trigger Pod replacement when source files change. Lense's topology contains internal Service URLs for server-side access and localhost links for port-forwarded browsers.

Sense's init container copies the initial config into its PVC **only when config.json is absent**. This keeps configuration edits and history made through the UI writable and persistent across Pod replacements. Editing the seed ConfigMap does not overwrite an existing Sense configuration: apply changes through the Sense/Lense configuration UI. New installations use the new seed. The initial five sources match Example 03.

Postgres history and Sense configuration survive Pod restarts. Broker queues/retained messages and simulator state are disposable; publishers refresh the sample after restarts. Deployments use one replica; do not scale Sense or Lense without assigning unique MQTT client identities and revisiting collection/consumer ownership. `Recreate` prevents overlapping collectors and conflicting writers to a ReadWriteOnce volume. Probes report service reachability, not end-to-end data freshness; check Lense and the smoke test for the latter.

## Verify and troubleshoot

```sh
bash scripts/kubernetes-smoke.sh
kubectl -n otio logs deployment/protocol-lab --tail=100
kubectl -n otio logs deployment/sense --tail=100
kubectl -n otio get events --sort-by=.lastTimestamp
```

The smoke test runs from an application Pod, tests native reads and broker samples, checks all five collected sources, and verifies Lense has stored numeric telemetry. Run it from Bash (Git Bash/WSL on Windows). `Pending` PVCs usually indicate a missing StorageClass/provisioner; `ImagePullBackOff` usually means images were not loaded into kind or registry credentials are missing. Lense's init container waits for Postgres; inspect Postgres first if Lense remains in `Init`.

CI renders the example, performs server-side validation in a temporary kind cluster, builds/loads the images, waits for rollout and runs the smoke test. Local rendering was verified; no local Docker daemon or selected Kubernetes context was available for a live rollout.

## Remove

`kubectl delete -k examples/05-kubernetes` removes this example **including its namespace and PVCs**, so treat it as a destructive demo reset. To pause while preserving storage, scale the deployments to zero instead:

```sh
kubectl -n otio scale deployment --all --replicas=0
```

For the dedicated local cluster, `kind delete cluster --name otio` removes the cluster and its stored demo data.

References: [Kustomize](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/), [probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/), [persistent volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).
