#!/bin/sh
# Reproduit le cas « generic-1 » (capture du 2026-10-16).
docker run -d --name telemach-exited-generic-1 --user 1000:1000 alpine:3.24 sh -c echo démarrage du service; echo "erreur : configuration invalide" >&2; exit 1
