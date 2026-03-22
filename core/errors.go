package core

import "errors"

var (
	ErrRepositoryWithSameIdentity = errors.New("Tried to add two repositories with the same identity")
	ErrModuleNotFound = errors.New("Module could not be found")
	ErrSubConfigNotFound = errors.New("No subconfig with the given prefix was found")
)
