#!/usr/bin/env bash
# Hook SessionStart : injecte LESSON-LEARNED.md dans le contexte de la session
# (principe V de la constitution — relecture obligatoire à chaque session).
set -euo pipefail

project_dir="${CLAUDE_PROJECT_DIR:-$(pwd)}"
lessons_file="$project_dir/LESSON-LEARNED.md"

if [[ ! -f "$lessons_file" ]]; then
  jq -n '{systemMessage: "⚠️ LESSON-LEARNED.md introuvable à la racine du projet (principe V de la constitution)."}'
  exit 0
fi

jq -n --rawfile lessons "$lessons_file" '{
  systemMessage: "📘 LESSON-LEARNED.md chargé dans le contexte (principe V).",
  hookSpecificOutput: {
    hookEventName: "SessionStart",
    additionalContext: ("RAPPEL (principe V de la constitution) : relis les leçons ci-dessous et applique leurs règles préventives pendant toute la session. Tout nouvel incident doit être consigné dans LESSON-LEARNED.md.\n\n" + $lessons)
  }
}'
