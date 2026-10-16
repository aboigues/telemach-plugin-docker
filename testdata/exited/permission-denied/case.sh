#!/bin/sh
# Reproduit le cas « permission-denied » (capture du 2026-10-16).
docker run -d --name telemach-exited-permission-denied --user 1000:1000 alpine:3.24 sh -c echo démarrage; echo x > /etc/app.conf
