// Package media handles image conversions, notably extracting a usable JPEG
// from camera RAW files without any external dependency.
package media

import (
	"bytes"
	"errors"
	"fmt"
	"image/jpeg"
)

// ErrNoPreview is returned when a RAW file carries no decodable JPEG preview.
var ErrNoPreview = errors.New("no embedded JPEG preview found")

// rawExtensions are the camera RAW file extensions we attempt to convert.
var rawExtensions = map[string]bool{
	".rw2": true, // Panasonic Lumix
	".raw": true,
	".cr2": true, ".cr3": true, // Canon
	".nef": true, // Nikon
	".arw": true, // Sony
	".orf": true, // Olympus
	".raf": true, // Fujifilm
	".dng": true, // Adobe / generic
	".pef": true, // Pentax
}

// IsRawExtension reports whether ext (lower-case, with leading dot) is a known
// RAW format.
func IsRawExtension(ext string) bool {
	return rawExtensions[ext]
}

// ExtractPreviewJPEG finds the largest embedded JPEG preview in a RAW file's
// bytes, then decodes and re-encodes it into a clean standalone JPEG.
//
// RAW files (TIFF-based: RW2, CR2, NEF, DNG…) embed one or more JPEG previews.
// We scan for JPEG start markers, validate each with the standard decoder, and
// keep the one with the largest dimensions — the full-size preview rather than
// the thumbnail.
func ExtractPreviewJPEG(data []byte) ([]byte, error) {
	const marker = "\xff\xd8\xff" // JPEG SOI + first APP marker byte

	bestStart, bestArea := -1, 0
	for i := 0; i < len(data); {
		rel := bytes.Index(data[i:], []byte(marker))
		if rel < 0 {
			break
		}
		start := i + rel

		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data[start:]))
		if err == nil {
			if area := cfg.Width * cfg.Height; area > bestArea {
				bestArea, bestStart = area, start
			}
		}
		i = start + len(marker)
	}

	if bestStart < 0 {
		return nil, ErrNoPreview
	}

	img, err := jpeg.Decode(bytes.NewReader(data[bestStart:]))
	if err != nil {
		return nil, fmt.Errorf("decoding embedded preview: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("re-encoding preview: %w", err)
	}
	return buf.Bytes(), nil
}
