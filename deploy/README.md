# Deploy the playground

The production origin is **https://sumi.fmap.dev**, on SSH host `i`. The application
checkout lives at `/root/repos/sumi`. Merge both the sumi deployment PR and the
`lesscap/skylab` Caddy site PR before the first public deployment.

```text
Internet → shared Caddy (TLS, :443)
  /api/*, /healthz → 172.17.0.1:5810 → api:8080
  everything else → 172.17.0.1:5811 → web:80
```

The Go API and static frontend are separate containers. They store no submitted
source and need no database or application volumes. The shared Caddy owns TLS;
the web container only serves static files. API paths must be preserved when
proxying (`handle`, not `handle_path`).

## Build and start

Requires Docker with Compose v2 or later, Git, and outbound access to GitHub,
Docker Hub and the npm registry. Images build on the host architecture. There is
no registry publication step.

On host `i`:

```sh
mkdir -p /root/repos
git clone https://github.com/bencode/sumi.git /root/repos/sumi
cd /root/repos/sumi
git rev-parse HEAD
docker compose -f deploy/compose.yml config --quiet
docker compose -f deploy/compose.yml up -d --build --wait --wait-timeout 120
docker compose -f deploy/compose.yml ps
curl --fail http://172.17.0.1:5810/healthz
curl --fail -o /dev/null http://172.17.0.1:5811/
```

Ports 5810 and 5811 bind to the Docker bridge address, rather than the public
interface. The shared Caddy container must be able to reach that address. For a
different bridge, set `SUMI_BIND_IP` for every Compose command. For local builds:

```sh
SUMI_BIND_IP=127.0.0.1 docker compose -f deploy/compose.yml up -d --build --wait --wait-timeout 120
```

Local port 5811 serves static files only. To use the complete playground, put a
local reverse proxy in front of both services with the same path routing as the
production site. Vite remains the simpler option for local development; see the
[repository README](../README.md#local-playground).

## Shared Caddy

The site block is maintained in
[`lesscap/skylab: machines/i/Caddyfile`](https://github.com/lesscap/skylab/blob/main/machines/i/Caddyfile).
The running container is `i-caddy-1`; its configuration is bind-mounted from
`/root/skylab/machines/i/Caddyfile`. Keep other sites and their certificates intact.

After the app is healthy, fetch the merged skylab configuration into a candidate
file. Check its diff against the live file: it must only add the sumi site. If
there are other differences, reconcile them before applying the candidate.

```sh
cd /root/skylab
git fetch origin main
candidate=$(mktemp /tmp/sumi-caddy.XXXXXX)
git show origin/main:machines/i/Caddyfile > "$candidate"
diff -u machines/i/Caddyfile "$candidate"
# diff exits 1 when the expected site addition is present.
docker exec -i i-caddy-1 caddy validate --config - --adapter caddyfile < "$candidate"
```

Back up the live configuration, then write in place to preserve the inode used by
Docker's file bind mount. Do not replace it with `mv` or update it with `git pull`
while that mount is active. Apply only after validation succeeds:

```sh
backup_dir="/root/sumi-deploy-backup/$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$backup_dir"
cp /root/skylab/machines/i/Caddyfile "$backup_dir/Caddyfile"
cat "$candidate" > /root/skylab/machines/i/Caddyfile
docker exec i-caddy-1 caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
docker exec i-caddy-1 caddy reload --config /etc/caddy/Caddyfile --adapter caddyfile
rm "$candidate"
```

This updates the live file without advancing the skylab checkout's HEAD. Retain
that fact when inspecting its Git status. Do not restart the shared Caddy or
change its global configuration. If validation or reload fails, restore the
backup in place and validate/reload again before proceeding.

## Verify

```sh
curl --fail https://sumi.fmap.dev/healthz
curl --fail -o /dev/null https://sumi.fmap.dev/
curl --fail https://sumi.fmap.dev/api/render \
  -H 'Content-Type: application/json' \
  --data '{"source":"sequenceDiagram\nAlice->>Bob: Hello"}'
docker compose -f /root/repos/sumi/deploy/compose.yml logs --tail 50
```

In a browser, check all 17 initial previews, edit a diagram, switch themes, reset
an example, and open the feature checklist (a new GitHub tab). Unknown assets
under `/assets/` should return 404; built assets have immutable cache headers and
HTML uses `no-cache`. The API retains its 64 KiB request limit and JSON errors.

## Update and rollback

Record the running revision before updating a clean sumi checkout:

```sh
cd /root/repos/sumi
git rev-parse HEAD
git pull --ff-only
docker compose -f deploy/compose.yml up -d --build --wait --wait-timeout 120
```

Repeat the checks above after each update. Builds happen before replacement of
the running containers; this is a small Compose deployment, without a promise of
zero downtime. Each service has a health check, resource limits, and rotated
container logs. A Go shutdown has 15 seconds to finish in-flight requests.

To roll back the application, check out the recorded good commit and rebuild:

```sh
cd /root/repos/sumi
git checkout <previous-good-commit>
docker compose -f deploy/compose.yml up -d --build --wait --wait-timeout 120
```

For a Caddy rollback, restore the saved file **in place**, validate, then reload:

```sh
cat /root/sumi-deploy-backup/<timestamp>/Caddyfile > /root/skylab/machines/i/Caddyfile
docker exec i-caddy-1 caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
docker exec i-caddy-1 caddy reload --config /etc/caddy/Caddyfile --adapter caddyfile
```

If stopping the application is necessary, use `docker compose -f
deploy/compose.yml down` from its checkout. Do not stop unrelated stacks or prune
shared Docker resources.
