#!/bin/sh
# Reproduit le cas « not-executable » (capture du 2026-10-16).
docker run -d --name telemach-exited-not-executable --user 1000:1000 alpine:3.24 /etc/hostname
