#!/bin/sh
# Reproduit le cas « address-retry-then-other » (capture du 2026-10-16).
docker run -d --name telemach-exited-address-retry-then-other --user 1000:1000 alpine:3.24 sh -c echo "bind: address already in use, nouvelle tentative"; i=0; while [ $i -lt 15 ]; do echo "traitement $i"; i=$((i+1)); done; echo "erreur : base de données injoignable"; exit 1
