package exited

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/aboigues/telemach-plugin-docker/internal/dockerread"
)

// fakeLister simule le démon : candidats de « docker ps » et états de « docker inspect ».
type fakeLister struct {
	cands     []dockerread.Candidate
	byID      map[string]dockerread.Container
	inspected [][]string
}

func (f *fakeLister) ListCandidates() ([]dockerread.Candidate, error) { return f.cands, nil }

func (f *fakeLister) Inspect(ids ...string) ([]dockerread.Container, error) {
	f.inspected = append(f.inspected, ids)
	var out []dockerread.Container
	for _, id := range ids {
		c, ok := f.byID[id]
		if !ok {
			return nil, &dockerread.CLIError{Code: 1, Stderr: "Error: No such object: " + id}
		}
		out = append(out, c)
	}
	return out, nil
}

var tBase = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

func ctr(id string, code int, finishedMin int, status string) dockerread.Container {
	c := dockerread.Container{ID: id, Name: "c-" + id, ExitCode: code, Status: status, Created: tBase}
	if finishedMin >= 0 {
		c.FinishedAt = tBase.Add(time.Duration(finishedMin) * time.Minute)
	}
	return c
}

func TestSelectOrderAndFilters(t *testing.T) {
	f := &fakeLister{
		cands: []dockerread.Candidate{
			{ID: "a", State: "exited", HasCode: true, Code: 1},
			{ID: "zero", State: "exited", HasCode: true, Code: 0}, // écarté sans inspection
			{ID: "b", State: "exited", HasCode: true, Code: 137},
			{ID: "created-err", State: "created"},
			{ID: "created-ok", State: "created"}, // simple docker create : écarté après inspection
			{ID: "loop", State: "restarting", HasCode: true, Code: 2},
		},
		byID: map[string]dockerread.Container{
			"a": ctr("a", 1, 5, "exited"),
			"b": ctr("b", 137, 30, "exited"),
			"created-err": func() dockerread.Container {
				c := ctr("created-err", 127, -1, "created")
				c.Error = "exec: x: not found"
				c.Created = tBase.Add(20 * time.Minute)
				return c
			}(),
			"created-ok": ctr("created-ok", 0, -1, "created"),
			"loop":       ctr("loop", 2, 10, "restarting"),
		},
	}
	got, err := Select(f)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	if want := []string{"b", "created-err", "loop", "a"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("ordre = %v, attendu %v (dernier arrêt d'abord)", ids, want)
	}
	if len(f.inspected) != 1 {
		t.Errorf("un seul appel à inspect attendu : %v", f.inspected)
	}
	for _, id := range f.inspected[0] {
		if id == "zero" {
			t.Error("un conteneur sorti avec 0 ne doit pas être inspecté")
		}
	}
}

func TestSelectInspectsAtMost25(t *testing.T) {
	f := &fakeLister{byID: map[string]dockerread.Container{}}
	for i := 0; i < 60; i++ {
		id := fmt.Sprintf("c%02d", i)
		f.cands = append(f.cands, dockerread.Candidate{ID: id, State: "exited", HasCode: true, Code: 1})
		f.byID[id] = ctr(id, 1, i, "exited")
	}
	if _, err := Select(f); err != nil {
		t.Fatal(err)
	}
	if n := len(f.inspected[0]); n != MaxInspect {
		t.Errorf("%d conteneurs inspectés, attendu %d", n, MaxInspect)
	}
}

func TestFind(t *testing.T) {
	f := &fakeLister{
		cands: []dockerread.Candidate{{ID: "a", Name: "api-web", State: "exited", HasCode: true, Code: 1}},
		byID: map[string]dockerread.Container{
			"api-web": ctr("a", 1, 5, "exited"),
			"ok":      ctr("o", 0, 5, "exited"),
			"run":     ctr("r", 0, -1, "running"),
		},
	}
	if c, err := Find(f, "api-web"); err != nil || c.ID != "a" {
		t.Errorf("Find(api-web) = %+v, %v", c, err)
	}
	var nf *NotFoundError
	if _, err := Find(f, "api-wbe"); !errors.As(err, &nf) || !reflect.DeepEqual(nf.Suggestions, []string{"api-web"}) {
		t.Errorf("introuvable avec suggestion attendu : %v", err)
	}
	if _, err := Find(f, "ok"); !errors.Is(err, ErrNotFailed) {
		t.Errorf("arrêté avec 0 : ErrNotFailed attendu, obtenu %v", err)
	}
	if _, err := Find(f, "run"); !errors.Is(err, ErrRunning) {
		t.Errorf("en cours : ErrRunning attendu, obtenu %v", err)
	}
}

// De simples « docker create » récents ne doivent pas prendre la place de vrais arrêts.
func TestSelectCreatedDoNotCrowdOut(t *testing.T) {
	f := &fakeLister{byID: map[string]dockerread.Container{}}
	for i := 0; i < 40; i++ {
		id := fmt.Sprintf("cr%02d", i)
		f.cands = append(f.cands, dockerread.Candidate{ID: id, State: "created"})
		f.byID[id] = ctr(id, 0, -1, "created")
	}
	f.cands = append(f.cands, dockerread.Candidate{ID: "vrai", State: "exited", HasCode: true, Code: 1})
	f.byID["vrai"] = ctr("vrai", 1, 5, "exited")
	got, err := Select(f)
	if err != nil || len(got) != 1 || got[0].ID != "vrai" {
		t.Errorf("le vrai arrêt en erreur doit être retenu : %+v, %v", got, err)
	}
	if n := len(f.inspected[0]); n != MaxInspect {
		t.Errorf("%d inspectés, attendu %d", n, MaxInspect)
	}
}
