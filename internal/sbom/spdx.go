package sbom

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Document struct {
	SPDXID            string         `json:"SPDXID"`
	SPDXVersion       string         `json:"spdxVersion"`
	DataLicense       string         `json:"dataLicense"`
	Name              string         `json:"name"`
	DocumentNamespace string         `json:"documentNamespace"`
	CreationInfo      CreationInfo   `json:"creationInfo"`
	Packages          []Package      `json:"packages"`
	Relationships     []Relationship `json:"relationships"`
}

type CreationInfo struct {
	Created  string   `json:"created"`
	Creators []string `json:"creators"`
}

type Package struct {
	SPDXID               string `json:"SPDXID"`
	Name                 string `json:"name"`
	VersionInfo          string `json:"versionInfo,omitempty"`
	DownloadLocation     string `json:"downloadLocation"`
	FilesAnalyzed        bool   `json:"filesAnalyzed"`
	PackageSupplier      string `json:"supplier"`
	PackageCopyrightText string `json:"copyrightText"`
}

type Relationship struct {
	SPDXElementID      string `json:"spdxElementId"`
	RelationshipType   string `json:"relationshipType"`
	RelatedSPDXElement string `json:"relatedSpdxElement"`
}

type Module struct {
	Path    string
	Version string
}

type Options struct {
	Root      string
	Name      string
	Namespace string
	CreatedAt time.Time
}

func Build(options Options) (Document, error) {
	root := options.Root
	if root == "" {
		root = "."
	}
	modules, err := parseGoMod(filepath.Join(root, "go.mod"))
	if err != nil {
		return Document{}, err
	}
	if len(modules) == 0 {
		return Document{}, fmt.Errorf("go.mod does not declare a module")
	}

	projectName := options.Name
	if projectName == "" {
		projectName = modules[0].Path
	}
	createdAt := options.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	namespace := options.Namespace
	if namespace == "" {
		namespace = fmt.Sprintf("https://spdx.org/spdxdocs/%s-%s", sanitizeID(projectName), shortHash(modules[0].Path))
	}

	packages := make([]Package, 0, len(modules))
	relationships := []Relationship{{
		SPDXElementID:      "SPDXRef-DOCUMENT",
		RelationshipType:   "DESCRIBES",
		RelatedSPDXElement: packageID(modules[0].Path),
	}}
	for i, module := range modules {
		packages = append(packages, Package{
			SPDXID:               packageID(module.Path),
			Name:                 module.Path,
			VersionInfo:          module.Version,
			DownloadLocation:     downloadLocation(module),
			FilesAnalyzed:        false,
			PackageSupplier:      "NOASSERTION",
			PackageCopyrightText: "NOASSERTION",
		})
		if i > 0 {
			relationships = append(relationships, Relationship{
				SPDXElementID:      packageID(modules[0].Path),
				RelationshipType:   "DEPENDS_ON",
				RelatedSPDXElement: packageID(module.Path),
			})
		}
	}

	return Document{
		SPDXID:            "SPDXRef-DOCUMENT",
		SPDXVersion:       "SPDX-2.3",
		DataLicense:       "CC0-1.0",
		Name:              projectName,
		DocumentNamespace: namespace,
		CreationInfo: CreationInfo{
			Created:  createdAt.UTC().Format(time.RFC3339),
			Creators: []string{"Tool: oss-maintainer-kit"},
		},
		Packages:      packages,
		Relationships: relationships,
	}, nil
}

func JSON(document Document) ([]byte, error) {
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func parseGoMod(path string) ([]Module, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}
	defer file.Close()
	return parseGoModReader(file)
}

func parseGoModReader(reader io.Reader) ([]Module, error) {
	var module Module
	var requires []Module
	inRequireBlock := false
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := cleanLine(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "module ") {
			module.Path = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			continue
		}
		if line == "require (" {
			inRequireBlock = true
			continue
		}
		if inRequireBlock {
			if line == ")" {
				inRequireBlock = false
				continue
			}
			if dep, ok := parseRequire(line); ok {
				requires = append(requires, dep)
			}
			continue
		}
		if strings.HasPrefix(line, "require ") {
			if dep, ok := parseRequire(strings.TrimSpace(strings.TrimPrefix(line, "require "))); ok {
				requires = append(requires, dep)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if module.Path == "" {
		return nil, nil
	}
	sort.Slice(requires, func(i, j int) bool {
		return requires[i].Path < requires[j].Path
	})
	return append([]Module{module}, requires...), nil
}

func cleanLine(line string) string {
	if idx := strings.Index(line, "//"); idx >= 0 {
		line = line[:idx]
	}
	return strings.TrimSpace(line)
}

func parseRequire(line string) (Module, bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return Module{}, false
	}
	return Module{Path: parts[0], Version: parts[1]}, true
}

var invalidID = regexp.MustCompile(`[^A-Za-z0-9.-]+`)

func packageID(name string) string {
	return "SPDXRef-Package-" + sanitizeID(name)
}

func sanitizeID(value string) string {
	value = invalidID.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "unknown"
	}
	return value
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return fmt.Sprintf("%x", sum[:])[:12]
}

func downloadLocation(module Module) string {
	if module.Version == "" {
		return "NOASSERTION"
	}
	return "pkg:golang/" + module.Path + "@" + module.Version
}
