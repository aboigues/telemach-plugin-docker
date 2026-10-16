"""Réduit les questions Stack Overflow à : identifiant, strate, score, titre, lignes d'erreur.

Sorties : so/questions.tsv (une ligne par question, dédoublonnée) et so/texts.txt (titre et
blocs de code de chaque question, \\n échappés, même ordre) pour « cov text ».
"""
import glob
import html
import json
import re
import sys

SO = sys.argv[1]
ERR = re.compile(r"(?i)(error|denied|cannot|can't|failed|unable|no such|not found|invalid|conflict|"
                 r"refused|unauthorized|exited with|exit code|returned a non-zero|is not running|"
                 r"already in use|already allocated|not allowed|no space|permission)")
seen = {}
for path in sorted(glob.glob(f"{SO}/*.json")):
    stratum = path.rsplit("/", 1)[1].split("-")[0]
    for q in json.load(open(path))["items"]:
        qid = q["question_id"]
        if qid in seen:
            seen[qid]["strata"].add(stratum)
            continue
        body = q.get("body", "")
        codes = [html.unescape(re.sub(r"<[^>]+>", "", c)) for c in re.findall(r"<code>(.*?)</code>", body, re.S)]
        title = html.unescape(q["title"])
        lines = [l.strip() for c in codes for l in c.splitlines() if ERR.search(l)]
        if ERR.search(title):
            lines.insert(0, "[titre] " + title)
        seen[qid] = {"strata": {stratum}, "score": q["score"], "title": title,
                     "lines": lines, "text": "\n".join([title] + codes)}

def esc(s):
    return s.replace("\\", "\\\\").replace("\t", "\\t").replace("\n", "\\n")

with open(f"{SO}/questions.tsv", "w") as qf, open(f"{SO}/texts.txt", "w") as tf:
    for qid, q in seen.items():
        errs = " ¦ ".join(l[:160] for l in q["lines"][:4])
        qf.write(f"{qid}\t{'+'.join(sorted(q['strata']))}\t{q['score']}\t{esc(q['title'])}\t{esc(errs) or '-'}\n")
        tf.write(esc(q["text"]) + "\n")
print(len(seen), "questions", file=sys.stderr)
