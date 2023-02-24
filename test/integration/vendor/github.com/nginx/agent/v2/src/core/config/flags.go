/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"time"

	flag "github.com/spf13/pflag"
)

// Registrable registers a pflag
type Registrable interface {
	register(*flag.FlagSet)
}

type StringFlag struct {
	Name         string
	Usage        string
	DefaultValue string
}

type StringSliceFlag struct {
	Name         string
	Usage        string
	DefaultValue []string
}

type StringMapFlag struct {
	Name         string
	Usage        string
	DefaultValue map[string]string
}

type IntFlag struct {
	Name         string
	Usage        string
	DefaultValue int
}

type BoolFlag struct {
	Name         string
	Usage        string
	DefaultValue bool
}

type DurationFlag struct {
	Name         string
	Usage        string
	DefaultValue time.Duration
}

func (f *StringFlag) register(fs *flag.FlagSet) {
	fs.String(f.Name, f.DefaultValue, f.Usage)
}

func (f *StringSliceFlag) register(fs *flag.FlagSet) {
	fs.StringSlice(f.Name, f.DefaultValue, f.Usage)
}

func (f *StringMapFlag) register(fs *flag.FlagSet) {
	fs.StringToString(f.Name, f.DefaultValue, f.Usage)
}

func (f *IntFlag) register(fs *flag.FlagSet) {
	fs.Int(f.Name, f.DefaultValue, f.Usage)
}

func (f *BoolFlag) register(fs *flag.FlagSet) {
	fs.Bool(f.Name, f.DefaultValue, f.Usage)
}

func (f *DurationFlag) register(fs *flag.FlagSet) {
	fs.Duration(f.Name, f.DefaultValue, f.Usage)
}
