package margin

import "testing"

func TestValid(t *testing.T) {
	t.Parallel()
	if !Isolated.Valid() {
		t.Fatal("expected 'true', received 'false'")
	}
	if !Multi.Valid() {
		t.Fatal("expected 'true', received 'false'")
	}
	if Unset.Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
	if Unknown.Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
	if Type(137).Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
}

func TestValid(t *testing.T) {
	t.Parallel()
	if !Isolated.Valid() {
		t.Fatal("expected 'true', received 'false'")
	}
	if !Multi.Valid() {
		t.Fatal("expected 'true', received 'false'")
	}
	if Unset.Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
	if Unknown.Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
	if Type(137).Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
}
