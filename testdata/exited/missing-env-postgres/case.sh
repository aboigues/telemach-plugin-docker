#!/bin/sh
# Reproduit le cas « missing-env-postgres » (capture du 2026-10-16).
docker run -d --name telemach-exited-missing-env-postgres postgres:18-alpine
