# Traefik secrets

`dashboard-users.htpasswd` lives here at runtime and is **never committed**
(`.gitignore` excludes `deployments/docker/traefik/secrets/*.htpasswd`).

It is generated from the `TRAEFIK_DASHBOARD_AUTH` secret by
`scripts/gen-traefik-auth.sh`, which the CD pipeline runs on the remote host
before `docker compose up`.

To create one by hand:

```bash
htpasswd -nbB admin "$(openssl rand -base64 24)" \
  > deployments/docker/traefik/secrets/dashboard-users.htpasswd
chmod 600 deployments/docker/traefik/secrets/dashboard-users.htpasswd
```

If the file is missing, the `dashboard-auth` middleware fails and the
dashboard / Prometheus routers stay closed. That is intentional.
