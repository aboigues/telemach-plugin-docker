#!/bin/sh
# Reproduit le cas « stopped » (capture du 2026-10-16).
docker run -d --name telemach-exited-stopped --user 1000:1000 alpine:3.24 sleep 300 ; docker stop -t 1 telemach-exited-stopped
