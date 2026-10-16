package dockerread

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Candidate est une ligne de « docker ps » : un conteneur arrêté, en boucle de redémarrage ou
// créé sans avoir démarré. Le code n'est connu sans inspection que pour les deux premiers.
type Candidate struct {
	ID, Name, Image, State, Status, CreatedAt string
	Code                                      int
	HasCode                                   bool
}

// statusCode extrait le code de « Exited (1) 3 seconds ago » ou « Restarting (2) … ».
var statusCode = regexp.MustCompile(`^(?:Exited|Restarting) \((\d+)\)`)

type psLine struct {
	ID        string `json:"ID"`
	Names     string `json:"Names"`
	Image     string `json:"Image"`
	State     string `json:"State"`
	Status    string `json:"Status"`
	CreatedAt string `json:"CreatedAt"`
}

func (p psLine) candidate() Candidate {
	c := Candidate{ID: p.ID, Name: p.Names, Image: p.Image, State: p.State, Status: p.Status, CreatedAt: p.CreatedAt}
	if m := statusCode.FindStringSubmatch(p.Status); m != nil {
		c.Code, _ = strconv.Atoi(m[1])
		c.HasCode = true
	}
	return c
}

// EnvNames sont les noms des variables d'environnement d'un conteneur. Le décodage ne garde
// que la partie avant « = » : aucune valeur n'est conservée (FR-009).
type EnvNames []string

// UnmarshalJSON décode ["NOM=valeur", …] en ne gardant que les noms.
func (e *EnvNames) UnmarshalJSON(data []byte) error {
	var raw []string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	names := make(EnvNames, len(raw))
	for i, kv := range raw {
		names[i], _, _ = strings.Cut(kv, "=")
		raw[i] = "" // la valeur ne survit pas au décodage
	}
	*e = names
	return nil
}

// Container est l'état d'un conteneur, décodé de « docker inspect » (data-model.md). Les
// chaînes sont nettoyées et masquées ; les variables d'environnement réduites à leurs noms.
type Container struct {
	ID, Name, Image, Command string
	ExitCode                 int
	Created                  time.Time
	StartedAt, FinishedAt    time.Time
	OOMKilled                bool
	Error                    string
	Status                   string
	Restarting               bool
	RestartCount             int
	RestartPolicy            string
	MemoryLimit              int64
	LogDriver                string
	EnvNames                 EnvNames
	ComposeProject           string
	ComposeService           string
}

type inspectJSON struct {
	ID           string    `json:"Id"`
	Name         string    `json:"Name"`
	Path         string    `json:"Path"`
	Args         []string  `json:"Args"`
	RestartCount int       `json:"RestartCount"`
	Created      time.Time `json:"Created"`
	State        struct {
		Status     string    `json:"Status"`
		Restarting bool      `json:"Restarting"`
		OOMKilled  bool      `json:"OOMKilled"`
		ExitCode   int       `json:"ExitCode"`
		Error      string    `json:"Error"`
		StartedAt  time.Time `json:"StartedAt"`
		FinishedAt time.Time `json:"FinishedAt"`
	} `json:"State"`
	Config struct {
		Image  string            `json:"Image"`
		Env    EnvNames          `json:"Env"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	HostConfig struct {
		RestartPolicy struct {
			Name string `json:"Name"`
		} `json:"RestartPolicy"`
		Memory    int64 `json:"Memory"`
		LogConfig struct {
			Type string `json:"Type"`
		} `json:"LogConfig"`
	} `json:"HostConfig"`
}

func (j inspectJSON) container() Container {
	cmd := append([]string{j.Path}, j.Args...)
	return Container{
		ID:             j.ID,
		Name:           clean(strings.TrimPrefix(j.Name, "/")),
		Image:          clean(j.Config.Image),
		Command:        cleanArgs(cmd),
		ExitCode:       j.State.ExitCode,
		Created:        j.Created,
		StartedAt:      j.State.StartedAt,
		FinishedAt:     j.State.FinishedAt,
		OOMKilled:      j.State.OOMKilled,
		Error:          clean(j.State.Error),
		Status:         j.State.Status,
		Restarting:     j.State.Restarting,
		RestartCount:   j.RestartCount,
		RestartPolicy:  j.HostConfig.RestartPolicy.Name,
		MemoryLimit:    j.HostConfig.Memory,
		LogDriver:      j.HostConfig.LogConfig.Type,
		EnvNames:       j.Config.Env,
		ComposeProject: clean(j.Config.Labels["com.docker.compose.project"]),
		ComposeService: clean(j.Config.Labels["com.docker.compose.service"]),
	}
}

// LogExcerpt est la fin du journal d'un conteneur, nettoyée et masquée ligne à ligne.
type LogExcerpt struct {
	Lines       []string
	Unavailable bool   // le pilote de journalisation ne permet pas la lecture
	Reason      string // message de la CLI (masqué) quand le journal est illisible
}

// Event est un événement de conteneur (« docker events »).
type Event struct {
	Action, ID, Name, Image string
	ExitCode                int // -1 sans code (événement autre que « die »)
	TimeNano                int64
}

type eventJSON struct {
	Action string `json:"Action"`
	Actor  struct {
		ID         string            `json:"ID"`
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
	TimeNano int64 `json:"timeNano"`
}

func (j eventJSON) event() Event {
	e := Event{
		Action: j.Action, ID: j.Actor.ID, Name: clean(j.Actor.Attributes["name"]),
		Image: clean(j.Actor.Attributes["image"]), ExitCode: -1, TimeNano: j.TimeNano,
	}
	if s, ok := j.Actor.Attributes["exitCode"]; ok {
		if n, err := strconv.Atoi(s); err == nil {
			e.ExitCode = n
		}
	}
	return e
}
