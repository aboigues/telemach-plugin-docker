#!/bin/sh
# Reproduit le cas « command-not-found » (capture du 2026-10-16).
docker run -d --name telemach-exited-command-not-found --user 1000:1000 alpine:3.24 commande-inexistante
