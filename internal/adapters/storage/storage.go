// Package storage persists uploaded media. The Storage interface lets the
// backend be swapped (local disk now, object storage later) without touching
// callers.
package storage

import (
	"context"
	"errors"
	"io"
)

// ErrUnsupportedType is returned when an upload is not an accepted image.
var ErrUnsupportedType = errors.New("unsupported media type")

// ErrTooLarge is returned when an upload exceeds the size limit.
var ErrTooLarge = errors.New("file too large")

// Saved describes a stored file.
type Saved struct {
	// URL is the public path at which the file is served (e.g. /media/x.jpg).
	URL string
}

// Storage stores and removes media files.
type Storage interface {
	// Save stores the data from r (already known to be contentType) and
	// returns its public URL.
	Save(ctx context.Context, contentType string, r io.Reader) (Saved, error)

	// Delete removes a file previously returned by Save, addressed by its URL.
	// Deleting an unknown or external URL is a no-op.
	Delete(ctx context.Context, url string) error
}

// allowedExt maps accepted image content types to file extensions.
var allowedExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// allowedAudioExt maps accepted audio content types to file extensions.
var allowedAudioExt = map[string]string{
	"audio/mpeg":      ".mp3",
	"audio/mp3":       ".mp3",
	"audio/ogg":       ".ogg",
	"application/ogg": ".ogg",
	"audio/wav":       ".wav",
	"audio/x-wav":     ".wav",
	"audio/webm":      ".weba",
}

// ExtensionFor returns the file extension for an accepted image content type,
// or ErrUnsupportedType.
func ExtensionFor(contentType string) (string, error) {
	if ext, ok := allowedExt[contentType]; ok {
		return ext, nil
	}
	return "", ErrUnsupportedType
}

// AudioExtensionFor returns the file extension for an accepted audio content
// type, or ErrUnsupportedType.
func AudioExtensionFor(contentType string) (string, error) {
	if ext, ok := allowedAudioExt[contentType]; ok {
		return ext, nil
	}
	return "", ErrUnsupportedType
}

// extensionForAny returns the extension for an accepted image OR audio content
// type. Used by the storage backends, which persist both kinds of media.
func extensionForAny(contentType string) (string, error) {
	if ext, ok := allowedExt[contentType]; ok {
		return ext, nil
	}
	if ext, ok := allowedAudioExt[contentType]; ok {
		return ext, nil
	}
	return "", ErrUnsupportedType
}
