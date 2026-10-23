package semantic

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	legacyFixtureHistorical = "---types---\nproof#10000001 = Proof;\ntoken#10000002 = Token;\nresult#10000003 = Result;\n" +
		"---functions---\nns.withdraw#20000001 proof:Proof = Result;\n// LAYER 1\n"
	legacyFixtureCanonical = "---types---\nproof#10000001 = Proof;\ntoken#10000002 = Token;\nresult#10000003 = Result;\n" +
		"---functions---\nns.withdraw#20000002 token:Token = Result;\n// LAYER 2\n"
	legacyFixtureDefinition = "ns.withdraw#20000001 proof:Proof = Result;"
)

func legacyFixtureManifest(t *testing.T, dir string, layers map[int]string, legacy []ManifestLegacyMethod) Manifest {
	t.Helper()
	manifest := Manifest{
		CanonicalLayer: 2,
		Repository:     "https://github.com/example/schema.git",
		SourcePath:     "api.tl",
		Overlays:       []ManifestOverlay{},
		LegacyMethods:  legacy,
	}
	for _, layer := range []int{1, 2} {
		source := []byte(layers[layer])
		artifact, err := InspectLayerArtifact(source)
		if err != nil {
			t.Fatal(err)
		}
		file := fmt.Sprintf("layer-%d.tl", layer)
		if err := os.WriteFile(filepath.Join(dir, file), source, 0o600); err != nil {
			t.Fatal(err)
		}
		manifest.Layers = append(manifest.Layers, ManifestLayer{
			Layer:  layer,
			File:   file,
			Commit: strings.Repeat("c", 40),
			Blob:   artifact.GitBlob,
			SHA256: artifact.SHA256,
		})
	}
	return manifest
}

func TestLegacyMethodPromotesHistoricalDefinition(t *testing.T) {
	dir := t.TempDir()
	manifest := legacyFixtureManifest(t, dir, map[int]string{
		1: legacyFixtureHistorical,
		2: legacyFixtureCanonical,
	}, []ManifestLegacyMethod{{Name: "ns.withdrawLegacy", Definition: legacyFixtureDefinition}})
	universe, err := LoadManifestUniverse(manifest, dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	legacyKey := SemanticKey{Category: CategoryFunction, QName: "ns.withdrawLegacy"}
	modernKey := SemanticKey{Category: CategoryFunction, QName: "ns.withdraw"}
	if got := universe.ByWire[1][0x20000001]; got == nil || got.Key != legacyKey {
		t.Fatalf("historical wire binds to %+v, want %s", got, legacyKey)
	}
	if got := universe.ByWire[2][0x20000001]; got == nil || got.Key != legacyKey {
		t.Fatalf("canonical wire binds to %+v, want %s", got, legacyKey)
	}
	legacy := universe.Families[legacyKey]
	if legacy == nil || legacy.ByLayer[1] == nil || legacy.ByLayer[2] == nil {
		t.Fatalf("legacy family = %+v, want layers 1 and 2", legacy)
	}
	if legacy.ByLayer[1].SignatureShape != legacy.ByLayer[2].SignatureShape {
		t.Fatal("promotion changed the historical signature")
	}
	modern := universe.Families[modernKey]
	if modern == nil || modern.ByLayer[1] != nil || modern.ByLayer[2] == nil || modern.ByLayer[2].WireID != 0x20000002 {
		t.Fatalf("superseding family = %+v, want canonical-only 0x20000002", modern)
	}

	rendered, err := RenderCanonicalSchema(manifest, []byte(legacyFixtureCanonical), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"// Legacy: ns.withdrawLegacy#20000001\n",
		"ns.withdraw#20000002 token:Token = Result;",
		"ns.withdrawLegacy#20000001 proof:Proof = Result;",
	} {
		if !bytes.Contains(rendered, []byte(want)) {
			t.Fatalf("canonical schema is missing %q:\n%s", want, rendered)
		}
	}
}

func TestLegacyMethodRejectsUnreviewedShapes(t *testing.T) {
	for _, test := range []struct {
		name   string
		layers map[int]string
		legacy ManifestLegacyMethod
		want   string
	}{
		{
			name:   "DefinitionDiffers",
			layers: map[int]string{1: legacyFixtureHistorical, 2: legacyFixtureCanonical},
			legacy: ManifestLegacyMethod{Name: "ns.withdrawLegacy", Definition: "ns.withdraw#20000001 proof:Token = Result;"},
			want:   "layer 1 wire ID 0x20000001 differs from reviewed legacy definition",
		},
		{
			name:   "Stale",
			layers: map[int]string{1: strings.Replace(legacyFixtureHistorical, "#20000001", "#20000003", 1), 2: legacyFixtureCanonical},
			legacy: ManifestLegacyMethod{Name: "ns.withdrawLegacy", Definition: legacyFixtureDefinition},
			want:   "stale legacy method",
		},
		{
			name:   "CanonicalStillDefinesWire",
			layers: map[int]string{1: legacyFixtureHistorical, 2: strings.Replace(legacyFixtureHistorical, "LAYER 1", "LAYER 2", 1)},
			legacy: ManifestLegacyMethod{Name: "ns.withdrawLegacy", Definition: legacyFixtureDefinition},
			want:   "canonical layer 2 still defines legacy wire ID",
		},
		{
			name:   "NotSuperseded",
			layers: map[int]string{1: legacyFixtureHistorical, 2: strings.Replace(legacyFixtureCanonical, "ns.withdraw#", "ns.withdrawNext#", 1)},
			legacy: ManifestLegacyMethod{Name: "ns.withdrawLegacy", Definition: legacyFixtureDefinition},
			want:   `has no superseding method "ns.withdraw"`,
		},
		{
			name:   "NameCollision",
			layers: map[int]string{1: legacyFixtureHistorical, 2: legacyFixtureCanonical},
			legacy: ManifestLegacyMethod{Name: "ns.withdraw", Definition: legacyFixtureDefinition},
			want:   "does not differ from its official definition",
		},
		{
			name:   "OtherNamespace",
			layers: map[int]string{1: legacyFixtureHistorical, 2: legacyFixtureCanonical},
			legacy: ManifestLegacyMethod{Name: "other.withdrawLegacy", Definition: legacyFixtureDefinition},
			want:   "must be an identifier in the namespace",
		},
		{
			name:   "NotAFunction",
			layers: map[int]string{1: legacyFixtureHistorical, 2: legacyFixtureCanonical},
			legacy: ManifestLegacyMethod{Name: "ns.withdrawLegacy", Definition: "---types---\nproof#10000001 = Proof;"},
			want:   "must be exactly one RPC function",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			manifest := legacyFixtureManifest(t, dir, test.layers, []ManifestLegacyMethod{test.legacy})
			_, err := LoadManifestUniverse(manifest, dir, nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("legacy method error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestLegacyMethodManifestRejectsDuplicates(t *testing.T) {
	for _, test := range []struct {
		name   string
		legacy []ManifestLegacyMethod
		want   string
	}{
		{
			name: "Name",
			legacy: []ManifestLegacyMethod{
				{Name: "ns.withdrawLegacy", Definition: legacyFixtureDefinition},
				{Name: "ns.withdrawLegacy", Definition: "ns.withdraw#20000003 proof:Proof = Result;"},
			},
			want: `repeats name "ns.withdrawLegacy"`,
		},
		{
			name: "Wire",
			legacy: []ManifestLegacyMethod{
				{Name: "ns.withdrawLegacy", Definition: legacyFixtureDefinition},
				{Name: "ns.withdrawOld", Definition: legacyFixtureDefinition},
			},
			want: "wire ID 0x20000001 is already promoted",
		},
		{
			name:   "ImplicitID",
			legacy: []ManifestLegacyMethod{{Name: "ns.withdrawLegacy", Definition: "ns.withdraw proof:Proof = Result;"}},
			want:   "E_EXPLICIT_ID_REQUIRED",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseLegacyMethods(test.legacy); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("legacy manifest error = %v, want %q", err, test.want)
			}
		})
	}
}
