/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package crossplane

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type NgxToken struct {
	Value    string
	Line     int
	IsQuoted bool
	Error    error
}

type state int

const (
	skipSpace state = iota
	inWord
	inComment
	inVar
	inQuote
)

const TokenChanCap = 2048

//nolint:gochecknoglobals
var lexerFile = "lexer" // pseudo file name for use by parse errors

//nolint:gochecknoglobals
var tokChanCap = TokenChanCap // capacity of lexer token channel

// note: this is only used during tests, should not be changed
func SetTokenChanCap(size int) {
	tokChanCap = size
}
func Lex(reader io.Reader) chan NgxToken {
	tc := make(chan NgxToken, tokChanCap)
	go tokenize(reader, tc)
	return tc
}

//nolint:gocyclo,funlen,gocognit
func tokenize(reader io.Reader, tokenCh chan NgxToken) {
	token := strings.Builder{}
	tokenLine := 1
	tokenStartLine := 1

	lexState := skipSpace
	newToken := false
	dupSpecialChar := false
	readNext := true
	esc := false
	depth := 0
	var la, quote string

	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanRunes)

	emit := func(line int, quoted bool, err error) {
		tokenCh <- NgxToken{Value: token.String(), Line: line, IsQuoted: quoted, Error: err}
		token.Reset()
		lexState = skipSpace
	}

	for {
		if readNext {
			if !scanner.Scan() {
				break // done
			}

			la = scanner.Text()
			if isEOL(la) {
				tokenLine++
			}
		} else {
			readNext = true
		}

		// skip CRs
		if la == "\r" || la == "\\\r" {
			continue
		}

		if la == "\\" && !esc {
			esc = true
			continue
		}
		if esc {
			esc = false
			la = "\\" + la
		}

		switch lexState {
		case skipSpace:
			if !isSpace(la) {
				lexState = inWord
				newToken = true
				readNext = false // re-eval
				tokenStartLine = tokenLine
			}
			continue
		case inWord:
			if newToken {
				newToken = false
				if la == "#" {
					token.WriteString(la)
					lexState = inComment
					tokenStartLine = tokenLine
					continue
				}
			}

			if isSpace(la) {
				emit(tokenStartLine, false, nil)
				continue
			}

			// handle parameter expansion syntax (ex: "${var[@]}")
			if token.Len() > 0 && strings.HasSuffix(token.String(), "$") && la == "{" {
				token.WriteString(la)
				lexState = inVar
				dupSpecialChar = false
				continue
			}

			// if a quote is found, add the whole string to the token buffer
			if la == `"` || la == "'" {
				if token.Len() > 0 {
					// if a quote is inside a token, treat it like any other char
					token.WriteString(la)
				} else {
					// swallow quote and change state
					quote = la
					lexState = inQuote
					tokenStartLine = tokenLine
				}
				dupSpecialChar = false
				continue
			}

			// handle special characters that are treated like full tokens
			if la == "{" || la == "}" || la == ";" {
				// if token complete yield it and reset token buffer
				if token.Len() > 0 {
					emit(tokenStartLine, false, nil)
				}

				// only '}' can be repeated
				if dupSpecialChar && la != "}" {
					emit(tokenStartLine, false, &ParseError{
						File: &lexerFile,
						What: fmt.Sprintf(`unexpected "%s"`, la),
						Line: &tokenLine,
					})
					close(tokenCh)
					return
				}

				dupSpecialChar = true

				if la == "{" {
					depth++
				}
				if la == "}" {
					depth--
					// early exit if unbalanced braces
					if depth < 0 {
						emit(tokenStartLine, false, &ParseError{File: &lexerFile, What: `unexpected "}"`, Line: &tokenLine})
						close(tokenCh)
						return
					}
				}

				token.WriteString(la)
				// this character is a full token so emit it
				emit(tokenStartLine, false, nil)
				continue
			}

			dupSpecialChar = false
			token.WriteString(la)

		case inComment:
			if isEOL(la) {
				emit(tokenStartLine, false, nil)
				continue
			}
			token.WriteString(la)

		case inVar:
			token.WriteString(la)
			// this is using the same logic as the exiting lexer, but this is wrong since it does not terminate on token boundary
			if !strings.HasSuffix(token.String(), "}") && !isSpace(la) {
				continue
			}
			lexState = inWord

		case inQuote:
			if la == quote {
				emit(tokenStartLine, true, nil)
				continue
			}
			if la == "\\"+quote {
				la = quote
			}
			token.WriteString(la)
		}
	}

	if token.Len() > 0 {
		emit(tokenStartLine, lexState == inQuote, nil)
	}
	if depth > 0 {
		emit(tokenStartLine, false, &ParseError{File: &lexerFile, What: `unexpected end of file, expecting "}"`, Line: &tokenLine})
	}

	close(tokenCh)
}
