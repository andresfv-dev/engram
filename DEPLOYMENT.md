# Engram Cloud Deployment

This deployment runs Postgres 16 on an internal Docker network. `engram-cloud`
is reachable only by containers on `proxy_net`; Cloudflare Tunnel exposes the
public HTTPS hostname. No database or cloud ports are published on the host.

## Operations

Start or update the local image:

```bash
docker compose up -d --build
```

Stop services while preserving Postgres data:

```bash
docker compose stop
```

Inspect status and logs:

```bash
docker compose ps
docker compose logs -f cloud
```

The confidential `.env` file is mode `0600` and is intentionally ignored by
Git. It contains the database password, recovery-only legacy runtime token,
JWT secret, and managed-token pepper. Do not distribute the legacy token;
clients must use individual managed tokens.

## Managed Access

The first managed administrator is created once with:

```bash
docker compose run --rm cloud cloud bootstrap admin --username admin --issue-token initial-admin
```

The raw token is displayed only once. Store it in a mode `0600` credentials
file before closing the terminal. Log in at `https://engram.andresfvelasquez.dev/dashboard/login`
with a managed admin token, then use **Admin / Users** to create users, grant
projects, issue named tokens, disable users, and revoke tokens. Issuing a
replacement token and revoking the old token rotates access without restarting
the service; revocation takes effect on the next request.

Managed users are deny-by-default for project sync. This deployment keeps the
runtime project policy at `*` to support arbitrary project names, but every
managed user still needs an explicit project grant in the admin dashboard.

## Client Setup

On each client, use that user's managed token, never the deployment `.env`:

```bash
engram cloud config --server https://engram.andresfvelasquez.dev
export ENGRAM_CLOUD_TOKEN='managed-token-for-this-user'
engram cloud enroll my-project
engram sync --cloud --project my-project
```

## Backup

Create an application-consistent SQL backup while Postgres is running:

```bash
mkdir -p backups
docker compose exec -T postgres pg_dump -U engram engram_cloud > backups/engram-cloud-$(date +%F).sql
chmod 600 backups/engram-cloud-*.sql
```

Copy backups to encrypted off-host storage. To restore, stop `cloud`, then use
`psql` or `pg_restore` against the private `postgres` service. Test restores in
a separate environment before relying on a backup.
