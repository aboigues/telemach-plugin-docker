#!/bin/sh
# Cas RETRANSCRIT, non capturé : Docker Desktop émule les autres architectures (qemu),
# « exec format error » ne se reproduit pas sur le poste de capture (2026-10-16).
# Message de runc pour un exécutable compilé pour une autre architecture, code 255.
# Reproduction sans émulation : docker run --platform linux/arm64 <image arm64 seulement>
