/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package ingester

import (
	"bytes"
)

const (
	separator               = ' '
	stringEscapingCharacter = '"'
)

type messageFieldIterator struct {
	msg             []byte
	currentPosition int
}

func newMessageFieldIterator(data []byte) *messageFieldIterator {
	return &messageFieldIterator{
		msg:             data,
		currentPosition: 0,
	}
}

func (i *messageFieldIterator) HasNext() bool {
	return len(i.msg) != 0 && i.currentPosition <= len(i.msg)
}

func (i *messageFieldIterator) Next() []byte {
	if !i.HasNext() {
		return nil
	}

	seekStartingPosition := i.currentPosition

	// XXX this will not handle case with `"` in middle of a string as with current
	// protocol strings with `"` are imposilbe to be correctly parsed. Changes are needed
	// in protocol iself in order to enable such strings.
	if isNextFieldStringField(i.msg[i.currentPosition:]) && i.currentPosition+1 < len(i.msg) {
		stringFieldEnd := bytes.IndexByte(i.msg[i.currentPosition+1:], stringEscapingCharacter)
		if stringFieldEnd != -1 {
			seekStartingPosition += stringFieldEnd
		}
	}

	msgLen := bytes.IndexByte(i.msg[seekStartingPosition:], separator)
	if msgLen == -1 {
		msgLen = len(i.msg) - seekStartingPosition
	}
	next := i.msg[i.currentPosition : seekStartingPosition+msgLen]

	i.currentPosition = msgLen + 1 + seekStartingPosition
	return next
}

func isNextFieldStringField(data []byte) bool {
	return len(data) > 0 && data[0] == stringEscapingCharacter
}
