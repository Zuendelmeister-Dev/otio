# Example 06 — Helm deployment

The chart at [`deploy/helm/otio`](../../deploy/helm/otio) deploys the same six-service Protocol Lab stack as Example 05: native Presense simulators, Sense, Lense, MQTT, AMQP and Postgres. This is a single-replica lab chart, without Dispense or the legacy standalone Presense services.

## Prepare images and cluster

Install Helm (tested with 3.19.0), kubectl, Docker and kind. For an existing cluster, use its selected kubectl context and registry images instead. A default StorageClass or explicitly configured storage is required.

Run from the repository root:

```sh
kind create cluster --name otio
docker build -t otio/protocol-lab:local -f apps/protocol-lab/Dockerfile .
docker build -t otio/sense:local -f apps/sense/Dockerfile .
docker build -t otio/lense:local -f apps/lense/Dockerfile .
kind load docker-image --name otio otio/protocol-lab:local otio/sense:local otio/lense:local
kubectl config current-context
```

If the `otio` kind cluster already exists from Example 05, reuse it. The Helm example uses namespace `otio-helm` and release-prefixed resources so it can coexist with the Kustomize example. Release names must be at most 40 characters.

## Render and install

```sh
helm lint deploy/helm/otio --strict
helm template otio deploy/helm/otio -n otio-helm -f examples/06-helm/values-kind.yaml
helm upgrade --install otio deploy/helm/otio -n otio-helm --create-namespace -f examples/06-helm/values-kind.yaml --wait --timeout 5m
helm test otio -n otio-helm --logs
```

The Helm test checks Lense's Protocol Lab proxy, valid Sense configuration and an actual native OPC UA read. It does not certify physical-device interoperability. `helm template` output includes Secrets; do not publish output rendered with private credentials.

## Access and try the demo

Run each port-forward in its own terminal:

```sh
kubectl -n otio-helm port-forward service/otio-lense 8000:8000
kubectl -n otio-helm port-forward service/otio-sense 8100:8100
kubectl -n otio-helm port-forward service/otio-protocol-lab 8500:8500
```

Open http://localhost:8000 and use its Protocol Lab link. Direct UIs are http://localhost:8100 and http://localhost:8500/static/. Stop other forwards using these local ports first. If you change the Helm release name, replace `otio-` in these Service names.

Select Modbus TCP, RTU-over-TCP or OPC UA and read the simulator. The native localhost presets work because the request runs inside the simulator Pod. Broker connections use `tcp://otio-mqtt:1883` and `amqp://lab:lab@otio-rabbitmq:5672/`; topics are `lab/temperature` and `amq.topic/lab.temperature`. The five configured Sense sources appear in Lense.

## Values and registry deployments

The chart's [values.yaml](../../deploy/helm/otio/values.yaml) configures:

| Value | Purpose |
|---|---|
| `images.*` | Full image references for applications, infrastructure and init/test containers |
| `imagePullPolicy`, `imagePullSecrets` | Application pull policy and private registry access |
| `credentials.*` | Postgres and RabbitMQ credentials; AMQP URL generated and escaped consistently |
| `persistence.postgresSize`, `persistence.senseSize` | PVC capacities |
| `persistence.storageClass` | `null`: default class; empty string: no dynamic provisioner; name: explicit class |
| `persistence.retain` | Keep PVCs when uninstalling (default `true`) |
| `browser.*` | Browser-facing links in Lense; internal Service URLs are generated separately |

For a registry deployment, push the three application images, then use a private override file:

```yaml
images:
  lense: registry.example.com/otio/lense:release-1
  sense: registry.example.com/otio/sense:release-1
  protocolLab: registry.example.com/otio/protocol-lab:release-1
imagePullSecrets:
  - name: registry-auth
persistence:
  storageClass: your-storage-class
  postgresSize: 10Gi
credentials:
  postgresPassword: replace-with-your-password
  rabbitmqPassword: replace-with-your-password
```

Create the registry Secret in the target namespace before installation. Pass the override file with an additional `-f` argument. Keep private values out of Git. The chart stores credentials and the Sense seed in Secrets, but Helm release records also contain values. The chart does not integrate an external secret manager. Anonymous MQTT and unauthenticated demo UIs are intended for a trusted lab.

## Upgrade, rollback and persistence

```sh
helm upgrade otio deploy/helm/otio -n otio-helm -f examples/06-helm/values-kind.yaml --wait --timeout 5m
helm history otio -n otio-helm
helm rollback otio 1 -n otio-helm --wait --timeout 5m
```

Use the same private override files on every upgrade. Prefer new immutable image tags: rebuilding an unchanged tag alone does not roll Pods. Configuration checksums trigger Pod replacement when chart-managed config changes.

Sense copies its seed only if the PVC has no config.json. UI configuration/history survives upgrades; changes to the Helm seed do not overwrite existing Sense settings. Apply updates through the Sense/Lense UI. Changing credentials in values does not rotate an initialized Postgres database or update Sense's stored AMQP connection; perform those changes together. Rollback restores chart resources, not database contents or UI-managed configuration. StorageClass changes and PVC shrinking are not supported on existing claims.

Postgres and Sense use persistent volumes; broker messages and simulator state are disposable. PVCs have `helm.sh/resource-policy: keep` by default. Uninstall removes the workloads and leaves those PVCs and the namespace:

```sh
helm uninstall otio -n otio-helm
kubectl -n otio-helm get pvc
```

Retained PVCs can be reused by reinstalling the same release in the same namespace. Deleting them or the namespace destroys the stored demo data. Setting `persistence.retain=false` and upgrading before uninstall permits Helm to remove the claims too.

## Verification

`node scripts/helm-chart.test.js` requires Helm on PATH and checks rendering, configurable images, release-based connection names, escaped credentials, PVC behavior and invalid values. `HELM_BIN` can point to another Helm executable. CI also installs this example into kind and runs `helm test`; local live installation remains unverified without a running Docker/Kubernetes environment.

References: [Helm charts](https://helm.sh/docs/topics/charts/) and [chart tests](https://helm.sh/docs/topics/chart_tests/).
