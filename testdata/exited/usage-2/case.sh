#!/bin/sh
# Reproduit le cas « usage-2 » (capture du 2026-10-16).
docker run -d --name telemach-exited-usage-2 --user 1000:1000 alpine:3.24 sh -c echo "usage: app [--port PORT] FICHIER" >&2; exit 2
