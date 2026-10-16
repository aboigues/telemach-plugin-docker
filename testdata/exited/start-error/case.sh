#!/bin/sh
# Reproduit le cas « start-error » (capture du 2026-10-16).
docker run -d --name telemach-exited-start-error --user utilisateur-inexistant alpine:3.24 true
