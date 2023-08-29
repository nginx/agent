package sysutils

import (
	"fmt"
	"strings"
)

// FakeShell mocks shell command output and errors
type FakeShell struct {
	Output map[string]string
	Errors map[string]error
}

// Exec facilitates mocking shell command execution
func (f *FakeShell) Exec(cmd string, arg ...string) ([]byte, error) {
	key := strings.Join(append([]string{cmd}, arg...), " ")
	if err, ok := f.Errors[key]; ok {
		return nil, err
	}
	if out, ok := f.Output[key]; ok {
		return []byte(out), nil
	}
	return nil, fmt.Errorf("unexpected command %s", key)
}
