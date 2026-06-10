package ports

import "errors"

// ErrNotFound is returned by repositories when an entity does not exist.
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists is returned by repositories when a unique constraint
// (e.g. a username) is violated.
var ErrAlreadyExists = errors.New("already exists")
