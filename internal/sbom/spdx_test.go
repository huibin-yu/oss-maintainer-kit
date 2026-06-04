package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildCreatesSPDXDocumentFromGoMod(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, `module github.com/acme/demo

go 1.21

require (
	golang.org/x/mod v0.14.0
	golang.org/x/sync v0.5.0 // indirect
)
`)

	doc, err := Build(Options{
		Root:      root,
		Name:      "demo",
		Namespace: "https://example.com/sbom/demo",
		CreatedAt: time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc.SPDXVersion != "SPDX-2.3" || doc.DataLicense != "CC0-1.0" {
		t.Fatalf("unexpected document metadata: %#v", doc)
	}
	if len(doc.Packages) != 3 {
		t.Fatalf("packages = %#v", doc.Packages)
	}
	if doc.Packages[0].Name != "github.com/acme/demo" {
		t.Fatalf("root package first: %#v", doc.Packages)
	}
	if doc.Packages[1].Name != "golang.org/x/mod" || doc.Packages[2].Name != "golang.org/x/sync" {
		t.Fatalf("dependencies not sorted: %#v", doc.Packages)
	}
	if len(doc.Relationships) != 3 {
		t.Fatalf("relationships = %#v", doc.Relationships)
	}
	if doc.Relationships[1].RelationshipType != "DEPENDS_ON" {
		t.Fatalf("missing dependency relationship: %#v", doc.Relationships)
	}
}

func TestJSONProducesValidSPDXJSON(t *testing.T) {
	data, err := JSON(Document{
		SPDXID:      "SPDXRef-DOCUMENT",
		SPDXVersion: "SPDX-2.3",
		DataLicense: "CC0-1.0",
		Name:        "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["spdxVersion"] != "SPDX-2.3" {
		t.Fatalf("unexpected JSON: %s", data)
	}
}

func TestBuildRejectsMissingGoMod(t *testing.T) {
	_, err := Build(Options{Root: t.TempDir()})
	if err == nil {
		t.Fatal("expected missing go.mod error")
	}
}

func writeGoMod(t *testing.T, root, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
