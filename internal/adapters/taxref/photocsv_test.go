package taxref

import (
	"strings"
	"testing"
)

func TestParsePhotoCSV(t *testing.T) {
	const sample = "photo;groupe_taxonomique;nom_scientifique\n" +
		"P1000733.JPG;Amphibiens;Bombina variegata\n" +
		"Hyla meridionalis.jpg;Amphibiens;Hyla meridionalis\n" +
		"\n" + // blank line ignored
		"P1020736.RW2;Reptiles;Natrix maura\n"

	rows, err := parsePhotoCSV(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("parsePhotoCSV() error = %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3 (header + blank skipped)", len(rows))
	}
	if rows[0].Photo != "P1000733.JPG" || rows[0].ScientificName != "Bombina variegata" {
		t.Errorf("row 0 = %+v", rows[0])
	}
	if rows[1].ScientificName != "Hyla meridionalis" {
		t.Errorf("row 1 name = %q", rows[1].ScientificName)
	}
	if rows[2].Group != "Reptiles" {
		t.Errorf("row 2 group = %q, want Reptiles", rows[2].Group)
	}
}
