package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadURLInputAcceptsURLAndFile(t *testing.T) {
	direct, err := readURLInput("https://example.com/archive")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(direct, []string{"https://example.com/archive"}) {
		t.Fatalf("direct input = %#v", direct)
	}

	path := filepath.Join(t.TempDir(), "urls.txt")
	if err := os.WriteFile(path, []byte("# publications\nhttps://one.example/archive\n\nhttps://two.example/archive\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readURLInput(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://one.example/archive", "https://two.example/archive"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("file input = %#v, want %#v", got, want)
	}
}
