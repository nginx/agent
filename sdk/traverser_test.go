/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sdk

import (
	"errors"
	"testing"

	crossplane "github.com/nginxinc/nginx-go-crossplane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildSampleConfig constructs a minimal crossplane.Config with two top-level
// directives: "events" containing one nested directive, and "http" containing
// a nested "server" with a "listen" directive.
//
//	events {
//	    worker_connections 1024;
//	}
//	http {
//	    server {
//	        listen 80;
//	    }
//	}
func buildSampleConfig() *crossplane.Config {
	listen := &crossplane.Directive{Directive: "listen", Args: []string{"80"}}
	server := &crossplane.Directive{Directive: "server", Block: []*crossplane.Directive{listen}}
	http := &crossplane.Directive{Directive: "http", Block: []*crossplane.Directive{server}}

	workerConn := &crossplane.Directive{Directive: "worker_connections", Args: []string{"1024"}}
	events := &crossplane.Directive{Directive: "events", Block: []*crossplane.Directive{workerConn}}

	return &crossplane.Config{Parsed: []*crossplane.Directive{events, http}}
}

func TestCrossplaneConfigTraverse_TableDriven(t *testing.T) {
	type traverseTestCase struct {
		name        string
		cfg         *crossplane.Config
		cb          func(parent, current *crossplane.Directive) (bool, error)
		wantVisited []string
		wantCount   int
		wantErr     error
		assertFn    func(t *testing.T, visited []string, count int, err error)
	}

	sampleCfg := buildSampleConfig()
	wantErr := errors.New("callback boom")
	wantTopErr := errors.New("top error")

	tests := []traverseTestCase{
		{
			name: "Test 1: VisitsEveryDirective",
			cfg:  sampleCfg,
			cb: func(_, current *crossplane.Directive) (bool, error) {
				return true, nil
			},
			wantVisited: []string{"events", "worker_connections", "http", "server", "listen"},
			assertFn: func(t *testing.T, visited []string, _ int, err error) {
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{"events", "worker_connections", "http", "server", "listen"}, visited)
			},
		},
		{
			name: "Test 2: StopsOnFalse",
			cfg:  sampleCfg,
			cb: func(_, current *crossplane.Directive) (bool, error) {
				return false, nil
			},
			wantCount: 1,
			assertFn: func(t *testing.T, _ []string, count int, err error) {
				require.NoError(t, err)
				assert.Equal(t, 1, count, "callback should be invoked exactly once before stopping")
			},
		},
		{
			name: "Test 3: StopsOnFalseInsideBlock",
			cfg:  sampleCfg,
			cb: func(_, current *crossplane.Directive) (bool, error) {
				if current.Directive == "server" {
					return false, nil
				}
				return true, nil
			},
			assertFn: func(t *testing.T, visited []string, _ int, err error) {
				require.NoError(t, err)
				assert.NotContains(t, visited, "listen")
				assert.Contains(t, visited, "server")
			},
		},
		{
			name: "Test 4: PropagatesError",
			cfg:  sampleCfg,
			cb: func(_, current *crossplane.Directive) (bool, error) {
				if current.Directive == "worker_connections" {
					return false, wantErr
				}
				return true, nil
			},
			wantErr: wantErr,
			assertFn: func(t *testing.T, _ []string, count int, err error) {
				require.ErrorIs(t, err, wantErr)
				assert.Greater(t, count, 0)
			},
		},
		{
			name: "Test 5: PropagatesErrorFromTopLevel",
			cfg:  sampleCfg,
			cb: func(parent, _ *crossplane.Directive) (bool, error) {
				if parent == nil {
					return false, wantTopErr
				}
				return true, nil
			},
			wantErr: wantTopErr,
			assertFn: func(t *testing.T, _ []string, _ int, err error) {
				assert.ErrorIs(t, err, wantTopErr)
			},
		},
		{
			name: "Test 6: EmptyConfig",
			cfg:  &crossplane.Config{Parsed: nil},
			cb: func(_, _ *crossplane.Directive) (bool, error) {
				return true, nil
			},
			assertFn: func(t *testing.T, _ []string, called int, err error) {
				require.NoError(t, err)
				assert.Equal(t, 0, called, "callback should not be invoked for empty config")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visited := make([]string, 0)
			count := 0
			err := CrossplaneConfigTraverse(tt.cfg, func(parent, current *crossplane.Directive) (bool, error) {
				count++
				if current != nil {
					visited = append(visited, current.Directive)
				}
				return tt.cb(parent, current)
			})
			if tt.assertFn != nil {
				tt.assertFn(t, visited, count, err)
			}
		})
	}
}

func TestCrossplaneConfigTraverse_DeepNesting(t *testing.T) {
	leaf := &crossplane.Directive{Directive: "e"}
	cur := leaf
	for _, name := range []string{"d", "c", "b", "a"} {
		cur = &crossplane.Directive{Directive: name, Block: []*crossplane.Directive{cur}}
	}
	cfg := &crossplane.Config{Parsed: []*crossplane.Directive{cur}}

	visited := make([]string, 0)
	err := CrossplaneConfigTraverse(cfg, func(_, current *crossplane.Directive) (bool, error) {
		visited = append(visited, current.Directive)
		return true, nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, visited)
}

func TestCrossplaneConfigTraverse_ParentArgPopulated(t *testing.T) {
	cfg := buildSampleConfig()

	pairs := make(map[string]string)
	err := CrossplaneConfigTraverse(cfg, func(parent, current *crossplane.Directive) (bool, error) {
		if parent != nil {
			pairs[current.Directive] = parent.Directive
		} else {
			pairs[current.Directive] = ""
		}
		return true, nil
	})
	require.NoError(t, err)
	assert.Equal(t, "", pairs["events"], "top-level directives have nil parent")
	assert.Equal(t, "", pairs["http"], "top-level directives have nil parent")
	assert.Equal(t, "events", pairs["worker_connections"])
	assert.Equal(t, "http", pairs["server"])
	assert.Equal(t, "server", pairs["listen"])
}

func TestCrossplaneConfigTraverseStr_ReturnsFirstMatch(t *testing.T) {
	cfg := buildSampleConfig()

	got := CrossplaneConfigTraverseStr(cfg, func(_, current *crossplane.Directive) string {
		if current.Directive == "listen" {
			return "found-listen"
		}
		return ""
	})

	assert.Equal(t, "found-listen", got)
}

func TestCrossplaneConfigTraverseStr_ReturnsTopLevelMatchEarly(t *testing.T) {
	cfg := buildSampleConfig()

	visited := 0
	got := CrossplaneConfigTraverseStr(cfg, func(_, current *crossplane.Directive) string {
		visited++
		if current.Directive == "events" {
			return "got-events"
		}
		return ""
	})

	assert.Equal(t, "got-events", got)
	assert.Equal(t, 1, visited, "should stop after first non-empty match")
}

func TestCrossplaneConfigTraverseStr_NoMatchReturnsEmpty(t *testing.T) {
	cfg := buildSampleConfig()

	got := CrossplaneConfigTraverseStr(cfg, func(_, _ *crossplane.Directive) string {
		return ""
	})

	assert.Empty(t, got)
}

func TestCrossplaneConfigTraverseStr_EmptyConfig(t *testing.T) {
	cfg := &crossplane.Config{Parsed: nil}

	called := false
	got := CrossplaneConfigTraverseStr(cfg, func(_, _ *crossplane.Directive) string {
		called = true
		return "x"
	})

	assert.Empty(t, got)
	assert.False(t, called)
}

func TestCrossplaneConfigTraverseStr_SearchesAcrossSiblings(t *testing.T) {
	first := &crossplane.Directive{Directive: "first"}
	matchTarget := &crossplane.Directive{Directive: "target"}
	second := &crossplane.Directive{Directive: "second", Block: []*crossplane.Directive{matchTarget}}
	cfg := &crossplane.Config{Parsed: []*crossplane.Directive{first, second}}

	got := CrossplaneConfigTraverseStr(cfg, func(_, current *crossplane.Directive) string {
		if current.Directive == "target" {
			return "hit"
		}
		return ""
	})

	assert.Equal(t, "hit", got)
}
