/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package crossplane

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// nolint:gochecknoglobals
var (
	hasMagic           = regexp.MustCompile(`[*?[]`)
	osOpen             = func(path string) (io.Reader, error) { return os.Open(path) }
	ErrPrematureLexEnd = errors.New("premature end of file")
)

type blockCtx []string

func (c blockCtx) key() string {
	return strings.Join(c, ">")
}

type fileCtx struct {
	path string
	ctx  blockCtx
}

type parser struct {
	configDir       string
	options         *ParseOptions
	handleError     func(*Config, error)
	includes        []fileCtx
	included        map[string]int
	includeEdges    map[string][]string
	includeInDegree map[string]int
}

// ParseOptions determine the behavior of an NGINX config parse.
type ParseOptions struct {
	// An array of directives to skip over and not include in the payload.
	IgnoreDirectives []string

	// If an error is found while parsing, it will be passed to this callback
	// function. The results of the callback function will be set in the
	// PayloadError struct that's added to the Payload struct's Errors array.
	ErrorCallback func(error) interface{}

	// If specified, use this alternative to open config files
	Open func(path string) (io.Reader, error)

	// Glob will return a matching list of files if specified
	Glob func(path string) ([]string, error)

	// If true, parsing will stop immediately if an error is found.
	StopParsingOnError bool

	// If true, include directives are used to combine all of the Payload's
	// Config structs into one.
	CombineConfigs bool

	// If true, only the config file with the given filename will be parsed
	// and Parse will not parse files included files.
	SingleFile bool

	// If true, comments will be parsed and added to the resulting Payload.
	ParseComments bool

	// If true, add an error to the payload when encountering a directive that
	// is unrecognized. The unrecognized directive will not be included in the
	// resulting Payload.
	ErrorOnUnknownDirectives bool

	// If true, checks that directives are in valid contexts.
	SkipDirectiveContextCheck bool

	// If true, checks that directives have a valid number of arguments.
	SkipDirectiveArgsCheck bool
}

// Parse parses an NGINX configuration file.
//
//nolint:funlen
func Parse(filename string, options *ParseOptions) (*Payload, error) {
	payload := &Payload{
		Status: "ok",
		Errors: []PayloadError{},
		Config: []Config{},
	}
	if options.Glob == nil {
		options.Glob = filepath.Glob
	}

	handleError := func(config *Config, err error) {
		var line *int
		if e, ok := err.(*ParseError); ok {
			line = e.Line
		}
		cerr := ConfigError{Line: line, Error: err}
		perr := PayloadError{Line: line, Error: err, File: config.File}
		if options.ErrorCallback != nil {
			perr.Callback = options.ErrorCallback(err)
		}

		const failedSts = "failed"
		config.Status = failedSts
		config.Errors = append(config.Errors, cerr)

		payload.Status = failedSts
		payload.Errors = append(payload.Errors, perr)
	}

	// Start with the main nginx config file/context.
	p := parser{
		configDir:   filepath.Dir(filename),
		options:     options,
		handleError: handleError,
		includes:    []fileCtx{{path: filename, ctx: blockCtx{}}},
		included:    map[string]int{filename: 0},
		// adjacency list where an edge exists between a file and the file it includes
		includeEdges: map[string][]string{},
		// number of times a file is included by another file
		includeInDegree: map[string]int{filename: 0},
	}

	for len(p.includes) > 0 {
		incl := p.includes[0]
		p.includes = p.includes[1:]

		file, err := p.openFile(incl.path)
		if err != nil {
			return nil, err
		}

		tokens := Lex(file)
		config := Config{
			File:   incl.path,
			Status: "ok",
			Errors: []ConfigError{},
			Parsed: Directives{},
		}
		parsed, err := p.parse(&config, tokens, incl.ctx, false)
		if err != nil {
			if options.StopParsingOnError {
				return nil, err
			}
			handleError(&config, err)
		} else {
			config.Parsed = parsed
		}

		payload.Config = append(payload.Config, config)
	}

	if p.isAcyclic() {
		return nil, errors.New("configs contain include cycle")
	}

	if options.CombineConfigs {
		return payload.Combined()
	}

	return payload, nil
}

func (p *parser) openFile(path string) (io.Reader, error) {
	open := osOpen
	if p.options.Open != nil {
		open = p.options.Open
	}
	return open(path)
}

// parse Recursively parses directives from an nginx config context.
// nolint:gocyclo,funlen,gocognit
func (p *parser) parse(parsing *Config, tokens <-chan NgxToken, ctx blockCtx, consume bool) (parsed Directives, err error) {
	var tokenOk bool
	// parse recursively by pulling from a flat stream of tokens
	for t := range tokens {
		if t.Error != nil {
			var perr *ParseError
			if errors.As(t.Error, &perr) {
				perr.File = &parsing.File
				return nil, perr
			}
			return nil, &ParseError{
				What:        t.Error.Error(),
				File:        &parsing.File,
				Line:        &t.Line,
				originalErr: t.Error,
			}
		}

		var commentsInArgs []string

		// we are parsing a block, so break if it's closing
		if t.Value == "}" && !t.IsQuoted {
			break
		}

		// if we are consuming, then just continue until end of context
		if consume {
			// if we find a block inside this context, consume it too
			if t.Value == "{" && !t.IsQuoted {
				_, _ = p.parse(parsing, tokens, nil, true)
			}
			continue
		}

		var fileName string
		if p.options.CombineConfigs {
			fileName = parsing.File
		}

		// the first token should always be an nginx directive
		stmt := &Directive{
			Directive: t.Value,
			Line:      t.Line,
			Args:      []string{},
			File:      fileName,
		}

		// if token is comment
		if strings.HasPrefix(t.Value, "#") && !t.IsQuoted {
			if p.options.ParseComments {
				comment := t.Value[1:]
				stmt.Directive = "#"
				stmt.Comment = &comment
				parsed = append(parsed, stmt)
			}
			continue
		}

		// parse arguments by reading tokens
		t, tokenOk = <-tokens
		if !tokenOk {
			return nil, &ParseError{
				What:        ErrPrematureLexEnd.Error(),
				File:        &parsing.File,
				Line:        &stmt.Line,
				originalErr: ErrPrematureLexEnd,
			}
		}
		for t.IsQuoted || (t.Value != "{" && t.Value != ";" && t.Value != "}") {
			if strings.HasPrefix(t.Value, "#") && !t.IsQuoted {
				commentsInArgs = append(commentsInArgs, t.Value[1:])
			} else {
				stmt.Args = append(stmt.Args, t.Value)
			}
			t, tokenOk = <-tokens
			if !tokenOk {
				return nil, &ParseError{
					What:        ErrPrematureLexEnd.Error(),
					File:        &parsing.File,
					Line:        &stmt.Line,
					originalErr: ErrPrematureLexEnd,
				}
			}
		}

		// if inside "map-like" block - add contents to payload, but do not parse further
		if len(ctx) > 0 {
			if _, ok := mapBodies[ctx[len(ctx)-1]]; ok {
				mapErr := analyzeMapBody(parsing.File, stmt, t.Value, ctx[len(ctx)-1])
				if mapErr != nil && p.options.StopParsingOnError {
					return nil, mapErr
				} else if mapErr != nil {
					p.handleError(parsing, mapErr)
					// consume invalid block
					if t.Value == "{" && !t.IsQuoted {
						_, _ = p.parse(parsing, tokens, nil, true)
					}
					continue
				}
				parsed = append(parsed, stmt)
				continue
			}
		}

		// consume the directive if it is ignored and move on
		if contains(p.options.IgnoreDirectives, stmt.Directive) {
			// if this directive was a block consume it too
			if t.Value == "{" && !t.IsQuoted {
				_, _ = p.parse(parsing, tokens, nil, true)
			}
			continue
		}

		// raise errors if this statement is invalid
		err = analyze(parsing.File, stmt, t.Value, ctx, p.options)

		if perr, ok := err.(*ParseError); ok && !p.options.StopParsingOnError {
			p.handleError(parsing, perr)
			// if it was a block but shouldn"t have been then consume
			if strings.HasSuffix(perr.What, ` is not terminated by ";"`) {
				if t.Value != "}" && !t.IsQuoted {
					_, _ = p.parse(parsing, tokens, nil, true)
				} else {
					break
				}
			}
			// keep on parsin'
			continue
		} else if err != nil {
			return nil, err
		}

		// prepare arguments - strip parentheses
		if stmt.Directive == "if" {
			stmt = prepareIfArgs(stmt)
		}

		// add "includes" to the payload if this is an include statement
		if !p.options.SingleFile && stmt.Directive == "include" {
			if len(stmt.Args) == 0 {
				return nil, &ParseError{
					What: fmt.Sprintf(`invalid number of arguments in "%s" directive in %s:%d`,
						stmt.Directive,
						parsing.File,
						stmt.Line,
					),
					File: &parsing.File,
					Line: &stmt.Line,
				}
			}

			pattern := stmt.Args[0]
			if !filepath.IsAbs(pattern) {
				pattern = filepath.Join(p.configDir, pattern)
			}

			// get names of all included files
			var fnames []string
			if hasMagic.MatchString(pattern) {
				fnames, err = p.options.Glob(pattern)
				if err != nil {
					return nil, err
				}
				sort.Strings(fnames)
			} else {
				// if the file pattern was explicit, nginx will check
				// that the included file can be opened and read
				if f, err := p.openFile(pattern); err != nil {
					perr := &ParseError{
						What: err.Error(),
						File: &parsing.File,
						Line: &stmt.Line,
					}
					if !p.options.StopParsingOnError {
						p.handleError(parsing, perr)
					} else {
						return nil, perr
					}
				} else {
					if c, ok := f.(io.Closer); ok {
						_ = c.Close()
					}
					fnames = []string{pattern}
				}
			}

			for _, fname := range fnames {
				// the included set keeps files from being parsed twice
				// TODO: handle files included from multiple contexts
				if _, ok := p.included[fname]; !ok {
					p.included[fname] = len(p.included)
					p.includes = append(p.includes, fileCtx{fname, ctx})
				}
				stmt.Includes = append(stmt.Includes, p.included[fname])
				// add edge between the current file and it's included file and
				// increase the included file's in degree
				p.includeEdges[parsing.File] = append(p.includeEdges[parsing.File], fname)
				p.includeInDegree[fname]++
			}
		}

		// if this statement terminated with "{" then it is a block
		if t.Value == "{" && !t.IsQuoted {
			stmt.Block = make(Directives, 0)
			inner := enterBlockCtx(stmt, ctx) // get context for block
			blocks, err := p.parse(parsing, tokens, inner, false)
			if err != nil {
				return nil, err
			}
			stmt.Block = append(stmt.Block, blocks...)
		}

		parsed = append(parsed, stmt)

		// add all comments found inside args after stmt is added
		for _, comment := range commentsInArgs {
			comment := comment
			parsed = append(parsed, &Directive{
				Directive: "#",
				Line:      stmt.Line,
				Args:      []string{},
				File:      fileName,
				Comment:   &comment,
			})
		}
	}

	return parsed, nil
}

// isAcyclic performs a topological sort to check if there are cycles created by configs' includes.
// First, it adds any files who are not being referenced by another file to a queue (in degree of 0).
// For every file in the queue, it will remove the reference it has towards its neighbors.
// At the end, if the queue is empty but not all files were once in the queue,
// then files still exist with references, and therefore, a cycle is present.
func (p *parser) isAcyclic() bool {
	// add to queue if file is not being referenced by any other file
	var queue []string
	for k, v := range p.includeInDegree {
		if v == 0 {
			queue = append(queue, k)
		}
	}
	fileCount := 0
	for len(queue) > 0 {
		// dequeue
		file := queue[0]
		queue = queue[1:]
		fileCount++

		// decrease each neighbor's in degree
		neighbors := p.includeEdges[file]
		for _, f := range neighbors {
			p.includeInDegree[f]--
			if p.includeInDegree[f] == 0 {
				queue = append(queue, f)
			}
		}
	}
	return fileCount != len(p.includeInDegree)
}
