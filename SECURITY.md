# Politique de sécurité

## Versions prises en charge

Seule la dernière version publiée de TELEMACH PLUGIN DOCKER reçoit des correctifs de sécurité.

## Signaler une vulnérabilité

Merci de **ne pas ouvrir de ticket public** pour une vulnérabilité.

Utilise le signalement privé de GitHub :
[Security › Report a vulnerability](https://github.com/aboigues/telemach-plugin-docker/security/advisories/new).

Indique si possible :

- la version concernée et ton système (Linux, macOS, WSL2) ;
- les étapes pour reproduire le problème ;
- l'impact estimé (fuite de secret, exécution de commande, injection de séquences de terminal…).

## Délais

- accusé de réception sous 7 jours ;
- première évaluation sous 14 jours ;
- correctif ou mesure d'atténuation dès que possible, avec mention du signalement si tu le souhaites.

## Engagements du projet

- Le plugin fonctionne entièrement en local, sans accès réseau.
- Il n'exécute aucune commande, à l'exception de `docker --version` lu une fois par
  l'intégration shell.
- Les valeurs sensibles sont masquées avant tout affichage ou toute écriture sur disque.
