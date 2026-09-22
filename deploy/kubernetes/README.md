# Kubernetes base

Reusable Kustomize resources for the Protocol Lab stack. Start with [Example 05](../../examples/05-kubernetes/README.md), which supplies demo credentials and instructions for kind and registry deployments.

An overlay must provide the `otio-credentials` Secret (keys are listed in Example 05), application images `otio/{protocol-lab,sense,lense}` and suitable storage provisioning for two PVCs. The base creates namespace `otio`; customize it consistently through Kustomize for another namespace. These are single-replica demo workloads, not an HA configuration.

Configuration seeds and browser/internal component URLs are under `config/`. Sense copies its seed only on first use of its persistent volume. See the example before changing or resetting configuration.
