package crossplane

import (
	"encoding/json"
	"fmt"
)

type ParseError struct {
	What        string
	File        *string
	Line        *int
	originalErr error
}

func (e *ParseError) Error() string {
	file := "(nofile)"
	if e.File != nil {
		file = *e.File
	}
	if e.Line != nil {
		return fmt.Sprintf("%s in %s:%d", e.What, file, *e.Line)
	}
	return fmt.Sprintf("%s in %s", e.What, file)
}

func (e *ParseError) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Error())
}

func (e *ParseError) Unwrap() error {
	return e.originalErr
}
