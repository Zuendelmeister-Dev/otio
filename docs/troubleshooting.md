# Troubleshooting

## Recommended start commands

Linux and macOS:

```bash
chmod +x scripts/*.sh
./scripts/start-demo.sh
```

Windows PowerShell:

```powershell
scripts/start-demo.ps1
```

Manual start on all platforms:

```bash
docker compose up --build
```

## Docker cannot resolve auth.docker.io

If the demo fails with an error similar to this:

```text
failed to fetch oauth token: Post "https://auth.docker.io/token":
dial tcp: lookup auth.docker.io: no such host
```

then the OT.io code is not the problem. Docker Desktop cannot resolve or reach Docker Hub.

The first local build needs access to Docker Hub because the Dockerfiles use official base images:

- `golang:1.22-alpine`
- `alpine:3.20`
- `postgres:16`
- `eclipse-mosquitto:2`

Run the preflight script from the repository root:

```powershell
scripts/preflight.ps1 -Pull
```

If this fails, try:

```powershell
docker pull alpine:3.20
docker pull golang:1.22-alpine
docker pull postgres:16
docker pull eclipse-mosquitto:2
```

If Docker still cannot resolve Docker Hub:

1. restart Docker Desktop
2. run `wsl --shutdown`
3. start Docker Desktop again
4. check VPN or proxy settings
5. check DNS resolution with `Resolve-DnsName auth.docker.io`

After the base images are available, start the demo:

```powershell
docker compose up --build
```

## Start the demo with one command

On Windows:

```powershell
scripts/start-demo.ps1
```

or:

```cmd
scripts\start-demo.cmd
```

The script checks Docker, pulls required base images and then starts the stack.


## Linux DNS checks

If Docker cannot resolve Docker Hub, test DNS from the host:

```bash
getent hosts auth.docker.io
```

If `getent` is not available:

```bash
nslookup auth.docker.io
```

or:

```bash
dig auth.docker.io
```

If these commands fail, check DNS, VPN, proxy and firewall settings.

## Linux Docker permissions

If Docker works only with `sudo`, either run the demo with `sudo` or add your user to the Docker group:

```bash
sudo usermod -aG docker "$USER"
```

Then log out and back in.

Check again:

```bash
docker info
```
