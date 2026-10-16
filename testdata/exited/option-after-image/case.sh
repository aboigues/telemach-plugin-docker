#!/bin/sh
# Reproduit le cas « option-after-image » (capture du 2026-10-16).
docker run -d --name telemach-exited-option-after-image nginxinc/nginx-unprivileged:1.30-alpine -p 8090:8080
