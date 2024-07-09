/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package crossplane

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

type BuildOptions struct {
	Indent      int
	Tabs        bool
	Header      bool
	Builders    []RegisterBuilder // handle specific directives
	extBuilders map[string]Builder
}

// RegisterBuilder is an option that can be used to add a builder to build NGINX configuration for custom directives.
type RegisterBuilder interface {
	applyBuildOptions(options *BuildOptions)
}

type registerBuilder struct {
	b          Builder
	directives []string
}

func (rb registerBuilder) applyBuildOptions(o *BuildOptions) {
	if o.extBuilders == nil {
		o.extBuilders = make(map[string]Builder)
	}

	for _, s := range rb.directives {
		o.extBuilders[s] = rb.b
	}
}

// BuildWithBuilder registers a builder to build the NGINX configuration for the given directives.
func BuildWithBuilder(b Builder, directives ...string) RegisterBuilder { //nolint:ireturn
	return registerBuilder{b: b, directives: directives}
}

// Builder is the interface implemented by types that can render a Directive
// as it appears in NGINX configuration files.
//
// Build writes the strings that represent the Directive and it's Block to the
// io.StringWriter returning any error encountered that caused the write to stop
// early. Build must not modify the Directive.
type Builder interface {
	Build(stmt *Directive) string
}

const MaxIndent = 100

//nolint:gochecknoglobals
var (
	marginSpaces = strings.Repeat(" ", MaxIndent)
	marginTabs   = strings.Repeat("\t", MaxIndent)
)

const header = `# This config was built from JSON using NGINX crossplane.
# If you encounter any bugs please report them here:
# https://github.com/nginxinc/crossplane/issues

`

// BuildFiles builds all of the config files in a crossplane.Payload and
// writes them to disk.
func BuildFiles(payload Payload, dir string, options *BuildOptions) error {
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		dir = cwd
	}

	for _, o := range options.Builders {
		o.applyBuildOptions(options)
	}

	for _, config := range payload.Config {
		path := config.File
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}

		// make directories that need to be made for the config to be built
		dirpath := filepath.Dir(path)
		if err := os.MkdirAll(dirpath, os.ModeDir|os.ModePerm); err != nil {
			return err
		}

		// build then create the nginx config file using the json payload
		var buf bytes.Buffer
		if err := Build(&buf, config, options); err != nil {
			return err
		}

		f, err := os.Create(path)
		if err != nil {
			return err
		}

		output := append(bytes.TrimRightFunc(buf.Bytes(), unicode.IsSpace), '\n')
		if _, err := f.Write(output); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Build creates an NGINX config from a crossplane.Config.
func Build(w io.Writer, config Config, options *BuildOptions) error {
	if options.Indent == 0 {
		options.Indent = 4
	}

	if options.Header {
		_, err := w.Write([]byte(header))
		if err != nil {
			return err
		}
	}

	if options.extBuilders == nil { // might be set if using BuildFiles
		for _, o := range options.Builders {
			o.applyBuildOptions(options)
		}
	}

	body := strings.Builder{}
	buildBlock(&body, nil, config.Parsed, 0, 0, options)

	bodyStr := body.String()
	if len(bodyStr) > 0 && bodyStr[len(bodyStr)-1] == '\n' {
		bodyStr = bodyStr[:len(bodyStr)-1]
	}

	_, err := w.Write([]byte(bodyStr))
	return err
}

//nolint:gocognit
func buildBlock(sb io.StringWriter, parent *Directive, block Directives, depth int, lastLine int, options *BuildOptions) {
	for i, stmt := range block {
		directive := Enquote(stmt.Directive)
		// if the this statement is a comment on the same line as the preview, do not emit EOL for this stmt
		if stmt.Line == lastLine && stmt.IsComment() {
			_, _ = sb.WriteString(" #")
			_, _ = sb.WriteString(*stmt.Comment)
			// sb.WriteString("\n")
			continue
		}

		if i != 0 || parent != nil {
			_, _ = sb.WriteString("\n")
		}

		_, _ = sb.WriteString(margin(options, depth))

		if options.extBuilders != nil {
			if ext, ok := options.extBuilders[directive]; ok {
				_, _ = sb.WriteString(ext.Build(stmt))
				continue
			}
		}

		if stmt.IsComment() {
			_, _ = sb.WriteString("#")
			_, _ = sb.WriteString(*stmt.Comment)
		} else {
			_, _ = sb.WriteString(directive)

			// special handling for if statements
			if directive == "if" {
				_, _ = sb.WriteString(" (")
				for i, arg := range stmt.Args {
					if i > 0 {
						_, _ = sb.WriteString(" ")
					}
					_, _ = sb.WriteString(Enquote(arg))
				}
				_, _ = sb.WriteString(")")
			} else {
				for _, arg := range stmt.Args {
					_, _ = sb.WriteString(" ")
					_, _ = sb.WriteString(Enquote(arg))
				}
			}

			if !stmt.IsBlock() {
				_, _ = sb.WriteString(";")
			} else {
				_, _ = sb.WriteString(" {")
				stmt := stmt
				buildBlock(sb, stmt, stmt.Block, depth+1, stmt.Line, options)
				_, _ = sb.WriteString("\n")
				_, _ = sb.WriteString(margin(options, depth))
				_, _ = sb.WriteString("}")
			}
		}

		lastLine = stmt.Line
	}
}

func margin(options *BuildOptions, depth int) string {
	indent := depth * options.Indent
	if indent < MaxIndent {
		if options.Tabs {
			return marginTabs[:depth]
		}
		return marginSpaces[:indent]
	}

	if options.Tabs {
		return strings.Repeat("\t", depth)
	}
	return strings.Repeat(" ", options.Indent*depth)
}

func Enquote(arg string) string {
	if !needsQuote(arg) {
		return arg
	}
	return strings.ReplaceAll(repr(arg), `\\`, `\`)
}

//nolint:gocyclo
func needsQuote(s string) bool {
	if s == "" {
		return true
	}

	// lexer should throw an error when variable expansion syntax
	// is messed up, but just wrap it in quotes for now I guess
	var char rune
	chars := escape(s)

	if len(chars) == 0 {
		return true
	}

	// get first rune
	char, off := utf8.DecodeRuneInString(chars)

	// arguments can't start with variable expansion syntax
	if unicode.IsSpace(char) || strings.ContainsRune("{};\"'", char) || strings.HasPrefix(chars, "${") {
		return true
	}

	chars = chars[off:]

	expanding := false
	var prev rune
	for _, c := range chars {
		char = c

		if prev == '\\' {
			prev = 0
			continue
		}
		if unicode.IsSpace(char) || strings.ContainsRune("{;\"'", char) {
			return true
		}

		if (expanding && (prev == '$' && char == '{')) || (!expanding && char == '}') {
			return true
		}

		if (expanding && char == '}') || (!expanding && (prev == '$' && char == '{')) {
			expanding = !expanding
		}

		prev = char
	}

	return expanding || char == '\\' || char == '$'
}

func escape(s string) string {
	if !strings.ContainsAny(s, "{}$;\\") {
		return s
	}

	sb := strings.Builder{}
	var pc, cc rune

	for _, r := range s {
		cc = r
		if pc == '\\' || (pc == '$' && cc == '{') {
			sb.WriteRune(pc)
			sb.WriteRune(cc)
			pc = 0
			continue
		}

		if pc == '$' {
			sb.WriteRune(pc)
		}
		if cc != '\\' && cc != '$' {
			sb.WriteRune(cc)
		}
		pc = cc
	}

	if cc == '\\' || cc == '$' {
		sb.WriteRune(cc)
	}

	return sb.String()
}

// BuildInto builds all of the config files in a crossplane.Payload and
// writes them to the Creator.
func BuildInto(payload *Payload, into Creator, options *BuildOptions) error {
	for _, config := range payload.Config {
		wc, err := into.Create(config.File)
		if err != nil {
			return err
		}
		if err := Build(wc, config, options); err != nil {
			return err
		}

		if err := wc.Close(); err != nil {
			return err
		}
	}

	return nil
}
