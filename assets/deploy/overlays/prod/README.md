# Prod overlay — public managed endpoint (single-node k3s)

Runs the full Optikk stack on one VM behind a public IP, with TLS on
`api.optikk.in` (query API + web) and `ingest.optikk.in` (OTLP gRPC/HTTP). This is
the managed endpoint `optikk onboard --api-url https://api.optikk.in` talks to.

What this overlay adds on top of `base/`:

- **TLS** via cert-manager + Let's Encrypt (HTTP-01). One SAN cert
  (`api`/`ingest.optikk.in`) → `optikk-tls` Secret, served on every TLS entrypoint by the
  single default `TLSStore`. Min TLS 1.2.
- **LoadBalancer** Traefik Service — k3s servicelb binds the node IP's 80/443/4317/4318.
- **Real secrets** from gitignored `secrets/*.env`, replacing the base placeholders
  (this is how the well-known `admin@optikk.dev / Password123!` seed gets rotated).

> Tenants are **not** listed here. Unlike `overlays/local` (which pins the `local` dev
> tenant), prod has no deterministic overlay tenant — every collector is provisioned at
> runtime by query's provisioner when a team signs up.

## Prerequisites

- A VM with **≥16 GiB RAM** (ClickHouse needs the headroom) and a public IP.
- DNS **A records** `api.optikk.in` and `ingest.optikk.in` → the VM's public IP.
- Ports 80, 443, 4317, 4318 open inbound. 80 must be reachable for the HTTP-01 challenge.

## 1. Install k3s (without its bundled Traefik)

k3s ships its own Traefik, servicelb, metrics-server, local-path storage, and CoreDNS.
We deploy our **own** Traefik, so disable k3s's to avoid a port-80/443 clash; keep the rest.

```bash
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--disable traefik" sh -
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml   # or copy it to ~/.kube/config
```

metrics-server (for the HPAs) and the `local-path` StorageClass come with k3s — nothing to
install.

## 2. App images

`ingest`/`query` must be pullable by containerd. Either make the `ghcr.io/optikklabs/*`
packages public (then no pull secret is needed), or import locally-built tars:

```bash
podman save -o images.tar ghcr.io/optikklabs/ingest:latest ghcr.io/optikklabs/query:latest
sudo k3s ctr images import images.tar
```

## 3. cert-manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
kubectl -n cert-manager rollout status deploy/cert-manager-webhook
```

## 4. Real secrets

Copy each template and fill in real values (these override the base placeholders and are
gitignored):

```bash
cd deploy/overlays/prod/secrets
for f in query mariadb clickhouse; do cp $f.env.example $f.env; done
# then edit query.env / mariadb.env / clickhouse.env:
#   jwt-secret:     openssl rand -hex 32
#   admin-password: a strong password (rotates the Password123! seed)
#   root-password / password: openssl rand -base64 24
```

The super-admin is seeded on **first boot only** from `query.env` — `EnsureSuperAdmin` is
idempotent and no-ops once the user exists, so rotating the admin password later means
updating it via the DB/API, not re-applying.

## 5. Deploy

```bash
kubectl apply -k deploy/overlays/prod
kubectl -n optikk rollout status statefulset/clickhouse statefulset/mariadb statefulset/mq
kubectl -n optikk wait --for=condition=available --timeout=180s deploy/ingest deploy/query deploy/traefik
```

> **Secrets fallback.** The overlay overrides secrets via `secretGenerator behavior: replace`
> (verified to override the plain base Secrets). If a future kustomize version regresses on
> that, drop the `secretGenerator` block and instead create the three secrets out-of-band —
> the repo's existing convention:
> ```bash
> kubectl create secret generic query-secret -n optikk \
>   --from-env-file=secrets/query.env --dry-run=client -o yaml | kubectl apply -f -
> ```

## 6. Issue the real certificate

The `Certificate` starts on `letsencrypt-staging` (untrusted, but avoids burning prod rate
limits while DNS/HTTP-01 is validated).

```bash
kubectl -n optikk get certificate optikk-tls -w   # wait for READY=True (staging)
```

Once staging is `Ready`, switch to a trusted cert: edit `cert-manager/certificate.yaml`,
set `issuerRef.name: letsencrypt-prod`, then:

```bash
kubectl apply -k deploy/overlays/prod
kubectl -n optikk delete secret optikk-tls    # force a fresh prod issuance
kubectl -n optikk get certificate optikk-tls -w
```

## 7. Smoke test (the stopwatch test)

```bash
curl -fsS https://api.optikk.in/health
# From a machine with a real tenant key (sign up first via the CLI):
optikk onboard --api-url https://api.optikk.in
```

Expected: signup → provisioned collector → OTLP-over-TLS snippet → first trace deep link.

## Deferred (not in this overlay)

- Nightly ClickHouse + MariaDB backups.
- Forced HTTP→HTTPS redirect (kept off so HTTP-01 stays simple; clients use https anyway).
- Offsite/object-storage backup durability.
