#!/bin/sh
# Reproduit le cas « option-after-image-debian » (capture du 2026-10-16).
docker run -d --name telemach-exited-option-after-image-debian nginx:1.30 -p 8090:80
