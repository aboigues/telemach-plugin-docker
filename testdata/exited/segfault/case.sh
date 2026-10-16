#!/bin/sh
# Reproduit le cas « segfault » (capture du 2026-10-16).
docker run -d --name telemach-exited-segfault --user 1000:1000 alpine:3.24 sh -c echo calcul; sh -c "kill -SEGV \$\$"; exit $?
