#!/bin/sh
# Reproduit le cas « address-in-use » (capture du 2026-10-16).
docker run -d --name telemach-exited-address-in-use --network container:telemach-exited-web nginxinc/nginx-unprivileged:1.30-alpine
