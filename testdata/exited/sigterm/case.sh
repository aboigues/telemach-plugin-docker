#!/bin/sh
# Reproduit le cas « sigterm » (capture du 2026-10-16).
docker run -d --name telemach-exited-sigterm --user 1000:1000 alpine:3.24 sh -c 'trap "echo arrêt demandé; exit 143" TERM; echo prêt; while :; do sleep 1; done' ; docker stop telemach-exited-sigterm
