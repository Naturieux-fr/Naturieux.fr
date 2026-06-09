// Package ports defines the interfaces between the application and adapters.
package ports

import "errors"

// ErrNotFound is returned by repositories when an entity does not exist.
var ErrNotFound = errors.New("not found")
