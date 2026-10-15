# Image telemachlearning/telemach : essayer TELEMACH sans installation (« check ») ou installer
# le plugin sur la machine hôte (« install »).
#
# Le contexte est préparé par GoReleaser (dockers_v2) : les binaires linux de la plateforme
# visée sous linux/<arch>/, et les dossiers de compilation darwin de dist/ (extra_files).
# Aucune compilation ici, aucun shell ni gestionnaire de paquets dans l'image (scratch),
# utilisateur non-root.
FROM scratch

ARG TARGETPLATFORM
ARG TARGETARCH

COPY $TARGETPLATFORM/telemach-image /usr/local/bin/telemach-image
COPY $TARGETPLATFORM/docker-telemach /opt/telemach/linux/docker-telemach
COPY dist/docker-telemach_darwin_${TARGETARCH}_*/docker-telemach /opt/telemach/darwin/docker-telemach

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/telemach-image"]
