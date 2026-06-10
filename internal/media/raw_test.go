package media_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/media"
)

// makeJPEG encodes a solid-colour image of the given size.
func makeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 80, G: 120, B: 60, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func TestExtractPreviewJPEG_PicksLargest(t *testing.T) {
	thumb := makeJPEG(t, 16, 12)     // small embedded thumbnail
	preview := makeJPEG(t, 320, 240) // full-size preview

	// A fake RAW: TIFF-ish header bytes, the thumbnail, padding, the preview.
	var raw bytes.Buffer
	raw.Write([]byte{0x49, 0x49, 0x55, 0x00, 0x18, 0x00, 0x00, 0x00}) // RW2-like magic
	raw.Write(thumb)
	raw.Write(bytes.Repeat([]byte{0x00}, 64))
	raw.Write(preview)

	jpg, err := media.ExtractPreviewJPEG(raw.Bytes())
	if err != nil {
		t.Fatalf("ExtractPreviewJPEG() error = %v", err)
	}

	cfg, err := jpeg.DecodeConfig(bytes.NewReader(jpg))
	if err != nil {
		t.Fatalf("result is not a valid JPEG: %v", err)
	}
	if cfg.Width != 320 || cfg.Height != 240 {
		t.Errorf("extracted %dx%d, want the 320x240 preview", cfg.Width, cfg.Height)
	}
}

func TestExtractPreviewJPEG_NoPreview(t *testing.T) {
	if _, err := media.ExtractPreviewJPEG([]byte("not a raw file at all")); err != media.ErrNoPreview {
		t.Errorf("error = %v, want ErrNoPreview", err)
	}
}

func TestIsRawExtension(t *testing.T) {
	for _, ext := range []string{".rw2", ".cr2", ".nef", ".dng"} {
		if !media.IsRawExtension(ext) {
			t.Errorf("IsRawExtension(%q) = false, want true", ext)
		}
	}
	for _, ext := range []string{".jpg", ".png", ".txt", ""} {
		if media.IsRawExtension(ext) {
			t.Errorf("IsRawExtension(%q) = true, want false", ext)
		}
	}
}
