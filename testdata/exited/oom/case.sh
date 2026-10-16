#!/bin/sh
# Reproduit le cas « oom » (capture du 2026-10-16).
docker run -d --name telemach-exited-oom --user 1000:1000 --memory 8m alpine:3.24 sh -c x=a; while :; do x=$x$x; done
