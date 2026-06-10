package taxref

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// PhotoCSVRow is one line of a photo-mapping CSV.
type PhotoCSVRow struct {
	Photo          string // file name within the photo directory
	Group          string // taxonomic group label (informative)
	ScientificName string // resolved against TAXREF
}

// ParsePhotoCSV reads a semicolon-separated mapping CSV with the header
// "photo;groupe_taxonomique;nom_scientifique".
func ParsePhotoCSV(path string) ([]PhotoCSVRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening CSV: %w", err)
	}
	defer func() { _ = f.Close() }()
	return parsePhotoCSV(f)
}

// parsePhotoCSV parses the CSV from a reader (testable).
func parsePhotoCSV(r io.Reader) ([]PhotoCSVRow, error) {
	reader := csv.NewReader(r)
	reader.Comma = ';'
	reader.FieldsPerRecord = -1 // tolerate ragged rows
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing CSV: %w", err)
	}

	rows := make([]PhotoCSVRow, 0, len(records))
	for i, rec := range records {
		if len(rec) < 3 {
			continue
		}
		photo := strings.TrimSpace(rec[0])
		name := strings.TrimSpace(rec[2])
		// Skip the header and blank lines.
		if i == 0 && strings.EqualFold(photo, "photo") {
			continue
		}
		if photo == "" || name == "" {
			continue
		}
		rows = append(rows, PhotoCSVRow{
			Photo:          photo,
			Group:          strings.TrimSpace(rec[1]),
			ScientificName: name,
		})
	}
	return rows, nil
}
