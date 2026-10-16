"""Mesure 2 de docs/couverture-catalogue.md : couverture pondérée du classement manuel.

Usage : python3 -I scripts/couverture/score.py docs/couverture/stackoverflow.tsv

Verdicts : C couverte ; G couverte de façon générique (étape RUN en échec, la cause est dans
l'application) ; N:<famille> non couverte ; O situation disparue (Compose v1, docker-py…) ;
X hors périmètre (erreur applicative, installation, Docker Desktop…).
"""
import collections
import sys

rows = [l.rstrip("\n").split("\t") for l in open(sys.argv[1]) if not l.startswith("#")]
if not rows:
    sys.exit("aucune question")

verdicts = collections.Counter()
by = collections.defaultdict(lambda: [0, 0, 0, 0])  # questions, couvertes, votes, votes couverts
gaps = collections.defaultdict(lambda: [0, 0])
for qid, strata, score, verdict in rows:
    kind = verdict.split(":")[0]
    if kind not in ("C", "G", "N", "O", "X"):
        sys.exit(f"verdict inconnu pour {qid} : {verdict}")
    verdicts[kind] += 1
    if kind not in ("C", "G", "N"):
        continue
    score = int(score)
    for st in strata.split("+") + ["total"]:
        b = by[st]
        b[0] += 1
        b[2] += score
        if kind != "N":
            b[1] += 1
            b[3] += score
    if kind == "N":
        g = gaps[verdict.split(":", 1)[1]]
        g[0] += 1
        g[1] += score

print("verdicts :", ", ".join(f"{k} {verdicts[k]}" for k in "CGNOX"))
for st in ("top", "recent", "compose", "total"):
    n, c, w, cw = by[st]
    if n == 0 or w == 0:
        sys.exit(f"strate vide : {st}")
    print(f"{st:8} {n:3} questions sur une erreur Docker, {c:3} couvertes ({100*c/n:.0f} %), "
          f"pondéré par les votes {100*cw/w:.0f} %")
print(f"dont {verdicts['G']} couvertes de façon générique (étape RUN)")
print("lacunes (questions, votes) :")
for fam, (n, w) in sorted(gaps.items(), key=lambda kv: (-kv[1][1], kv[0])):
    print(f"  {fam:36} {n:2} {w:5}")
