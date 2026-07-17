# Official image (not a custom pip-install build like hermes-bridge): the
# Hermes web Dashboard's frontend is a Vite/React bundle that has to be
# built with Node at image-creation time — that build output isn't part of
# the `hermes-agent` Python wheel a plain `pip install` pulls in, so a
# custom python:3.11-slim image (as this Dockerfile used to be) can never
# serve the dashboard, only the bare CLI. The official image ships the
# prebuilt web_dist alongside the CLI. Pin the tag; bump deliberately.
FROM nousresearch/hermes-agent:v2026.6.19

EXPOSE 9119

# --insecure: required by `hermes dashboard` to bind beyond 127.0.0.1 at
#     all (see `hermes dashboard --help`) — safe here because this port is
#     never published to the host (see deploy/docker-compose.yml), only
#     reachable from the `api` container's reverse proxy over the compose
#     network, same trust boundary the previous ttyd setup relied on.
# --no-open: no X server in a container; auto-launching a browser would
#     just fail/no-op.
# --skip-build is deliberately NOT passed — this is the official image, so
#     the dashboard's web_dist is already prebuilt; the flag exists for
#     custom builds without Node, which this no longer is.
#
# Path-prefix reverse proxying: `hermes dashboard` has no ttyd-style
# `-b`/base-path flag. Instead it natively honours an `X-Forwarded-Prefix`
# request header (see hermes_cli/web_server.py's `mount_spa`/
# `_normalise_prefix` — confirmed by reading the installed package source),
# rewriting the served index.html's asset URLs and injecting
# `window.__HERMES_BASE_PATH__` accordingly. Go's reverse proxy
# (internal/http/handlers/hermes_tui_handler.go) sets that header and
# strips the mount prefix before forwarding — both sides must agree.
CMD ["hermes", "dashboard", "--host", "0.0.0.0", "--insecure", "--no-open"]
