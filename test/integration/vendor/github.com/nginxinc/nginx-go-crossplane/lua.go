package crossplane

import (
	"fmt"
	"strings"
)

const (
	setByLuaBlock             = "set_by_lua_block"
	numArgsForSetByLuaBlock   = 2
	numArgsForOtherDirectives = 1
)

// Lua adds support for directives added to NGINX by the ngx_http_lua_module module.
//
// Lua implements the Lexer interface by tokenizing *_by_lua_block directives with a
// simple Lua parser. The Lua blocks will be placed into a token to be set as an
// argument on the Lua directive.
//
// Lua also implements the Builder interface by writing the *_by_lua_block
// directive's contents into the directive's block.
type Lua struct{}

// directiveNames returns a list of Lua module directive names used in NGINX configurations.
func (l *Lua) directiveNames() []string {
	return []string{
		"init_by_lua_block",
		"init_worker_by_lua_block",
		"exit_worker_by_lua_block",
		"set_by_lua_block",
		"content_by_lua_block",
		"server_rewrite_by_lua_block",
		"rewrite_by_lua_block",
		"access_by_lua_block",
		"header_filter_by_lua_block",
		"body_filter_by_lua_block",
		"log_by_lua_block",
		"balancer_by_lua_block",
		"ssl_client_hello_by_lua_block",
		"ssl_certificate_by_lua_block",
		"ssl_session_fetch_by_lua_block",
		"ssl_session_store_by_lua_block",
	}
}

// RegisterLexer registers a lexer for parsing Lua blocks.
func (l *Lua) RegisterLexer() RegisterLexer { //nolint:ireturn
	return LexWithLexer(l, l.directiveNames()...)
}

// Lex lexically analyzes the Lua blocks based on directives detected.
// It is used by the lexer to tokenize Lua content within configuration files.
//
//nolint:funlen,gocognit,gocyclo,nosec
func (l *Lua) Lex(s *SubScanner, matchedToken string) <-chan NgxToken {
	tokenCh := make(chan NgxToken)

	tokenDepth := 0

	go func() {
		defer close(tokenCh)
		var tok strings.Builder
		var inQuotes bool
		var quoteType string

		// special handling for'set_by_lua_block' directive
		// ignore potential hardcoded credentials linter warning for "set_by_lua_block"
		if matchedToken == setByLuaBlock /* #nosec G101 */ {
			arg := ""
			for {
				if !s.Scan() {
					return
				}
				next := s.Text()
				if isSpace(next) {
					if arg != "" {
						tokenCh <- NgxToken{Value: arg, Line: s.Line(), IsQuoted: false}
						break
					}

					for isSpace(next) {
						if !s.Scan() {
							return
						}
						next = s.Text()
					}
				}
				arg += next
			}
		}

		// check that Lua block starts correctly
		for {
			if !s.Scan() {
				return
			}
			next := s.Text()

			if !isSpace(next) {
				if next != "{" {
					lineno := s.Line()
					tokenCh <- NgxToken{Error: &ParseError{File: &lexerFile, What: `expected "{" to start lua block`, Line: &lineno}}
					return
				}
				tokenDepth++
				break
			}
		}

		// Grab everything in Lua block as a single token and watch for curly brace '{' in strings
		for {
			if !s.Scan() {
				return
			}

			next := s.Text()
			if err := s.Err(); err != nil {
				lineno := s.Line()
				tokenCh <- NgxToken{Error: &ParseError{File: &lexerFile, What: err.Error(), Line: &lineno}}
			}

			switch {
			case next == "{" && !inQuotes:
				tokenDepth++
				if tokenDepth > 1 { // not the first open brace
					tok.WriteString(next)
				}

			case next == "}" && !inQuotes:
				tokenDepth--
				if tokenDepth < 0 {
					lineno := s.Line()
					tokenCh <- NgxToken{Error: &ParseError{File: &lexerFile, What: `unexpected "}"`, Line: &lineno}}
					return
				}

				if tokenDepth > 0 { // not the last close brace for it to be 0
					tok.WriteString(next)
				}

				if tokenDepth == 0 {
					tokenCh <- NgxToken{Value: tok.String(), Line: s.Line(), IsQuoted: true}
					tokenCh <- NgxToken{Value: ";", Line: s.Line(), IsQuoted: false} // For an end to the Lua string based on the nginx bahavior
					// See: https://github.com/nginxinc/crossplane/blob/master/crossplane/ext/lua.py#L122C25-L122C41
					return
				}

			case next == `"` || next == "'":
				if !inQuotes {
					inQuotes = true
					quoteType = next
				} else if inQuotes && next == quoteType {
					inQuotes = false
				}
				tok.WriteString(next)

			default:
				// Expected first token is “{” to open a Lua block. If the first non-whitespace character is not “{”,
				// we are not starting Lua tokenization. This is crucial for cases like ‘server_name content_by_lua_block;’.
				// Without an opening “{”, ignore input until encountering a brace “{” with tokenDepth > 0.
				if isSpace(next) && tokenDepth == 0 {
					continue
				}

				// stricly check that first non space character is {
				if tokenDepth == 0 {
					tokenCh <- NgxToken{Value: next, Line: s.Line(), IsQuoted: false}
					return
				}
				tok.WriteString(next)
			}
		}
	}()

	return tokenCh
}

// RegisterBuilder registers a builder for generating Lua NGINX configuration.
func (l *Lua) RegisterBuilder() RegisterBuilder { //nolint:ireturn
	return BuildWithBuilder(l, l.directiveNames()...)
}

// Build generates Lua configurations based on the provided directive.
func (l *Lua) Build(stmt *Directive) string {
	if stmt.Directive == setByLuaBlock {
		if len(stmt.Args) < numArgsForSetByLuaBlock {
			return stmt.Directive
		}
		return fmt.Sprintf("%s %s {%s}", stmt.Directive, stmt.Args[0], stmt.Args[1])
	}
	if len(stmt.Args) < numArgsForOtherDirectives {
		return stmt.Directive
	}
	return fmt.Sprintf("%s {%s}", stmt.Directive, stmt.Args[0])
}
