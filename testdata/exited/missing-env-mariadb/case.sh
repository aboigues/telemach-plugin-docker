#!/bin/sh
# Reproduit le cas « missing-env-mariadb » (capture du 2026-10-16).
docker run -d --name telemach-exited-missing-env-mariadb mariadb:13.0
