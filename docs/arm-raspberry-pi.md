# ARM and Raspberry Pi Deployment

The Raspberry Pi lab deployment lives in:

```text
examples/02-raspberry-pi-arm-distributed
```

It deploys OT.io across four ARM nodes with an Ansible control container. The playbook copies a source bundle as the SSH user, renders node-specific configuration, then starts the matching Docker Compose stack on each Pi.

## Start Here

Use the full example guide:

- [Raspberry Pi ARM distributed example](../examples/02-raspberry-pi-arm-distributed/README.md)

The important moving parts are:

- `deploy/ansible/inventory.ini` for the non-secret fallback inventory
- `deploy/ansible/secrets/local-inventory.yml` for local lab credentials
- `deploy/ansible/templates/*.j2` for rendered Sense and Lense runtime configuration
- `examples/02-raspberry-pi-arm-distributed/node-stacks/*/docker-compose.yml` for the per-node Compose stacks

## Safety Notes

The example is meant for a trusted lab network. It exposes HTTP UIs, MQTT and Postgres ports directly to the LAN, and the Mosquitto demo config allows anonymous MQTT clients.

For a shared network, set `OTIO_CONFIG_WRITE_TOKEN` in the rendered environment file and use that token when the UI asks for the configuration token. Production use still needs proper authentication, TLS, broker credentials, secret management and network hardening.
