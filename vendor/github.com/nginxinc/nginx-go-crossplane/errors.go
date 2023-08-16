/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package crossplane

import (
	"encoding/json"
	"fmt"
)

type ParseError struct {
	What string
	File *string
	Line *int
	// Raw directive statement causing the parse error.
	Statement string
	// Block in which parse error occurred.
	BlockCtx    string
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
