package plugin

import (
	"bytes"
	"testing"
)

func TestWriteMetadata(t *testing.T) {
	var b bytes.Buffer
	if err := WriteMetadata(&b, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	want := `{"SchemaVersion":"0.1.0","Vendor":"TELEMACH","Version":"1.2.3","ShortDescription":"Explique en français les erreurs Docker et Docker Compose"}` + "\n"
	if got := b.String(); got != want {
		t.Errorf("métadonnées :\n%s\nattendu :\n%s", got, want)
	}
}
