# OPC UA certificates

In Lense, open **Certificates** and upload a named profile containing the client application certificate, its matching unencrypted PEM private key, and the trusted server leaf certificate. Use RSA keys of at least 2048 bits; the client certificate needs an Application URI in its subject alternative names. The OPC UA server must trust the client certificate.

Select the profile under **Protocol Lab → OPC UA** before reading a value or preparing a Sense deployment. The connection retains `?certificateProfile=NAME`. The gateway uses Basic256Sha256 with SignAndEncrypt and an anonymous user token. It pins the uploaded server certificate, checks both validity periods, and refuses to fall back to an unencrypted connection. Certificate profiles do not configure MQTT TLS or OPC UA user-certificate authentication. The older `opcua` HTTP demo is separate from native `lab-opcua-tcp`.

## Storage and access

Compose examples persist profiles in the `certificate-profiles` volume mounted at `OTIO_CERTIFICATE_DIR`. Keep this volume when recreating containers. The directory stores private keys, so restrict access and protect backups. Files are created with mode 0600 and the directory with mode 0700. Use HTTPS and set `OTIO_CONFIG_TOKEN` on Protocol Lab for uploads outside a trusted local environment; enter that token in the upload form. Metadata responses contain no private keys.

For other deployments, provision a persistent directory and set `OTIO_CERTIFICATE_DIR`. The general Kubernetes/Helm examples do not automatically provision writable certificate storage. The minimal `otio-sense` chart accepts `existingCertificateSecret` and mounts its profile JSON files read-only.

## Expiry and replacement

While Lense is open, it checks profiles every minute. A red banner appears when either certificate expires within 30 days, is expired, or is not yet valid. Unreadable profiles or an unavailable service produce a monitoring warning. Upload the same profile name to replace it; the next connection reads the replacement. This is manual rotation, with no automatic issuance, email notification, or replication to remote gateways.

## Deploying to a VM

Use **Deploy Sense** to specify the device address, VM SSH login and MQTT broker. Download the generated files and follow the displayed build, transfer and start commands. Images tagged `local` must be built; no prebuilt image download is implied. For a secured OPC UA connection, place your original `client.pem`, `client.key` and `server.pem` beside `prepare-certificate.py`, then run it as instructed. It creates the private profile locally; the commands transfer it separately and mount it read-only on the gateway. Helm instructions create a Kubernetes Secret from this file. Do not commit these certificates, keys or generated private profiles. Re-provision remote gateways when rotating certificates.
