# OT.io Helm chart

See [Example 06](../../../examples/06-helm/README.md) for image preparation, values, installation, tests, upgrades and storage lifecycle.

This self-contained chart packages the six-service Protocol Lab demonstration. Each release has its own Service names, configuration and volumes. No external chart dependencies are required. The configuration seeds in `files/` match the Kubernetes example; templates adapt their internal addresses to the release name.
