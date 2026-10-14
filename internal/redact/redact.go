package redact

import (
	"regexp"
	"strings"
)

// Mask remplace toute valeur sensible.
const Mask = "••••••"

// sensitiveName reconnaît un nom de variable sensible, sans distinction de casse.
const sensitiveName = `[A-Za-z0-9_.-]*(?i:PASSWORD|PASSWD|SECRET|TOKEN|KEY|CREDENTIAL|AUTH)[A-Za-z0-9_.-]*`

var (
	// NOM: "valeur" ou NOM: 'valeur' (YAML, valeur entre guillemets).
	quotedYAML = regexp.MustCompile(`(` + sensitiveName + `\s*:\s*)(["'])[^"'\n]*(["'])`)
	// NOM: valeur (clé YAML en début de ligne, éventuellement après « - », jusqu'à la fin
	// de ligne). L'ancrage évite de masquer du texte libre comme « unauthorized: … ».
	plainYAML = regexp.MustCompile(`(?m)^([ \t]*(?:-[ \t]+)?` + sensitiveName + `[ \t]*:[ \t]+)([^\s"'#][^\n]*)`)
	// NOM=valeur (option -e, fichier .env, liste environment).
	assignment = regexp.MustCompile(`(` + sensitiveName + `=)([^\s"']+|"[^"\n]*"|'[^'\n]*')`)
	// schéma://utilisateur:mot-de-passe@hôte
	urlCredentials = regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://[^\s:/@]+:)([^\s@/]+)(@)`)
	// En-tête Authorization: Bearer|Basic|Token <valeur>
	authorization = regexp.MustCompile(`(?i)(Authorization:\s*(?:Bearer|Basic|Token)\s+)(\S+)`)
	// Jetons reconnaissables par leur préfixe.
	knownTokens = regexp.MustCompile(`\b(?:ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{22,}|glpat-[A-Za-z0-9_-]{20,}|dckr_pat_[A-Za-z0-9_-]+|AKIA[0-9A-Z]{16})`)
	// Mot de passe passé en option : -p valeur / --password valeur / --password=valeur.
	passwordFlagEq = regexp.MustCompile(`(--password=)(\S+)`)
)

// Redact masque les valeurs sensibles d'un texte libre (sortie d'erreur, extrait de
// fichier, commande). Le nom de la variable est conservé pour rester pédagogique.
// La fonction est idempotente.
func Redact(s string) string {
	return redactLoginPassword(redactText(s))
}

// redactText applique les règles de masquage par motifs.
func redactText(s string) string {
	s = knownTokens.ReplaceAllString(s, Mask)
	s = authorization.ReplaceAllString(s, "${1}"+Mask)
	s = urlCredentials.ReplaceAllString(s, "${1}"+Mask+"${3}")
	s = quotedYAML.ReplaceAllStringFunc(s, func(m string) string {
		sub := quotedYAML.FindStringSubmatch(m)
		inner := strings.TrimSuffix(strings.TrimPrefix(m[len(sub[1]):], sub[2]), sub[3])
		if isReference(inner) {
			return m
		}
		return sub[1] + sub[2] + Mask + sub[3]
	})
	s = plainYAML.ReplaceAllStringFunc(s, func(m string) string {
		sub := plainYAML.FindStringSubmatch(m)
		if strings.Contains(sub[2], Mask) || isReference(sub[2]) || isFileVariable(sub[1]) {
			return m // déjà masqué, ou simple référence à une variable
		}
		return sub[1] + Mask
	})
	s = assignment.ReplaceAllStringFunc(s, func(m string) string {
		sub := assignment.FindStringSubmatch(m)
		if isReference(sub[2]) || isFileVariable(sub[1]) {
			return m
		}
		return sub[1] + Mask
	})
	return passwordFlagEq.ReplaceAllString(s, "${1}"+Mask)
}

// isFileVariable indique une variable *_FILE, qui contient le chemin d'un fichier de
// secret (convention des images officielles) et non le secret lui-même.
func isFileVariable(prefix string) bool {
	name := strings.TrimRight(strings.TrimLeft(prefix, " \t-"), " \t:=")
	upper := strings.ToUpper(name)
	// « secrets: » est aussi le mot-clé Compose qui liste des noms de secrets, pas des valeurs.
	return strings.HasSuffix(upper, "_FILE") || upper == "SECRETS"
}

// isReference indique si une valeur n'est qu'une référence à une variable (${NOM} ou
// $NOM), qui ne révèle aucun secret.
func isReference(v string) bool {
	v = strings.Trim(strings.TrimSpace(v), `"'`)
	return strings.HasPrefix(v, "$") && !strings.HasPrefix(v, "$$")
}

// redactLoginPassword masque la valeur suivant -p ou --password dans une commande
// docker login écrite sur une ligne.
func redactLoginPassword(s string) string {
	if !strings.Contains(s, "login") {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "login") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		masked := maskPasswordFlags(fields)
		changed := false
		for j := range fields {
			if fields[j] != masked[j] {
				changed = true
			}
		}
		if changed {
			lines[i] = strings.Join(masked, " ")
		}
	}
	return strings.Join(lines, "\n")
}

// RedactArgs masque les valeurs sensibles d'une liste d'arguments docker, sans modifier
// la liste d'origine.
func RedactArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = redactText(a)
	}
	return maskPasswordFlags(out)
}

// maskPasswordFlags masque la valeur qui suit --password, ou -p dans une commande login.
func maskPasswordFlags(args []string) []string {
	out := append([]string(nil), args...)
	isLogin := false
	for _, a := range out {
		if a == "login" {
			isLogin = true
		}
	}
	for i := 0; i < len(out)-1; i++ {
		switch out[i] {
		case "--password":
			out[i+1] = Mask
		case "-p":
			// -p signifie --publish pour docker run : on ne masque que pour docker login.
			if isLogin {
				out[i+1] = Mask
			}
		}
	}
	return out
}

var sensitiveNameOnly = regexp.MustCompile(`^` + sensitiveName + `$`)

// IsSensitiveName indique si un nom de variable désigne vraisemblablement un secret
// (PASSWORD, SECRET, TOKEN, KEY, CREDENTIAL, AUTH…, sans distinction de casse).
func IsSensitiveName(name string) bool {
	return sensitiveNameOnly.MatchString(name)
}
