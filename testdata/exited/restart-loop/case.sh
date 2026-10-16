#!/bin/sh
# Reproduit le cas « restart-loop » (capture du 2026-10-16).
docker run -d --name telemach-exited-restart-loop --user 1000:1000 --restart on-failure:3 alpine:3.24 sh -c 'echo tentative; exit 1'
