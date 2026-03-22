package util

import "reflect"

func TypeNameOf[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		// This happens when T itself is an interface type
		// and the zero value is a nil interface.
		return "<nil>"
	}
	return t.String()
}
