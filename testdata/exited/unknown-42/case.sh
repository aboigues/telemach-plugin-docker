#!/bin/sh
# Reproduit le cas « unknown-42 » (capture du 2026-10-16).
docker run -d --name telemach-exited-unknown-42 --user 1000:1000 alpine:3.24 sh -c echo traitement terminé; exit 42
