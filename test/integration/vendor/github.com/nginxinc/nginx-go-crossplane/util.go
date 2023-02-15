package crossplane

import (
	"fmt"
	"strings"
	"unicode"
)

type included struct {
	directive *Directive
	err       error
}

func contains(xs []string, x string) bool {
	for _, s := range xs {
		if s == x {
			return true
		}
	}
	return false
}

func isSpace(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func isEOL(s string) bool {
	return strings.HasSuffix(s, "\n")
}

func repr(s string) string {
	q := fmt.Sprintf("%q", s)
	for _, char := range s {
		if char == '"' {
			q = strings.ReplaceAll(q, `\"`, `"`)
			q = strings.ReplaceAll(q, `'`, `\'`)
			q = `'` + q[1:len(q)-1] + `'`
			return q
		}
	}
	return q
}

func validFlag(s string) bool {
	l := strings.ToLower(s)
	return l == "on" || l == "off"
}

// validExpr ensures an expression is enclused in '(' and ')' and is not empty.
func validExpr(d *Directive) bool {
	l := len(d.Args)
	b := 0
	e := l - 1

	return l > 0 &&
		strings.HasPrefix(d.Args[b], "(") &&
		strings.HasSuffix(d.Args[e], ")") &&
		((l == 1 && len(d.Args[b]) > 2) || // empty expression single arg '()'
			(l == 2 && (len(d.Args[b]) > 1 || len(d.Args[e]) > 1)) || // empty expression two args '(', ')'
			(l > 2))
}

// prepareIfArgs removes parentheses from an `if` directive's arguments.
func prepareIfArgs(d *Directive) *Directive {
	b := 0
	e := len(d.Args) - 1
	if len(d.Args) > 0 && strings.HasPrefix(d.Args[0], "(") && strings.HasSuffix(d.Args[e], ")") {
		d.Args[0] = strings.TrimLeftFunc(strings.TrimPrefix(d.Args[0], "("), unicode.IsSpace)
		d.Args[e] = strings.TrimRightFunc(strings.TrimSuffix(d.Args[e], ")"), unicode.IsSpace)
		if len(d.Args[0]) == 0 {
			b++
		}
		if len(d.Args[e]) == 0 {
			e--
		}
		d.Args = d.Args[b : e+1]
	}
	return d
}

// combineConfigs combines config files into one by using include directives.
func combineConfigs(old *Payload) (*Payload, error) {
	if len(old.Config) < 1 {
		return old, nil
	}

	status := old.Status
	if status == "" {
		status = "ok"
	}

	errors := old.Errors
	if errors == nil {
		errors = []PayloadError{}
	}

	combined := Config{
		File:   old.Config[0].File,
		Status: "ok",
		Errors: []ConfigError{},
		Parsed: Directives{},
	}

	for _, config := range old.Config {
		combined.Errors = append(combined.Errors, config.Errors...)
		if config.Status == "failed" {
			combined.Status = "failed"
		}
	}

	for incl := range performIncludes(old, combined.File, old.Config[0].Parsed) {
		if incl.err != nil {
			return nil, incl.err
		}
		combined.Parsed = append(combined.Parsed, incl.directive)
	}

	return &Payload{
		Status: status,
		Errors: errors,
		Config: []Config{combined},
	}, nil
}

func performIncludes(old *Payload, fromfile string, block Directives) chan included {
	c := make(chan included)
	go func() {
		defer close(c)
		for _, d := range block {
			dir := *d
			if dir.IsBlock() {
				nblock := Directives{}
				for incl := range performIncludes(old, fromfile, dir.Block) {
					if incl.err != nil {
						c <- incl
						return
					}
					nblock = append(nblock, incl.directive)
				}
				dir.Block = nblock
			}
			if !dir.IsInclude() {
				c <- included{directive: &dir}
				continue
			}
			for _, idx := range dir.Includes {
				if idx >= len(old.Config) {
					c <- included{
						err: &ParseError{
							What: fmt.Sprintf("include config with index: %d", idx),
							File: &fromfile,
							Line: &dir.Line,
						},
					}
					return
				}
				for incl := range performIncludes(old, old.Config[idx].File, old.Config[idx].Parsed) {
					c <- incl
				}
			}
		}
	}()
	return c
}
