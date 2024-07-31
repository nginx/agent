/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package crossplane

import "fmt"

// mapParameterMasks holds bit masks that define the behavior of the body of map-like directives.
// Some map directives have "special parameter" with different behaviors than the default.
type mapParameterMasks struct {
	specialParameterMasks map[string]uint
	defaultMasks          uint
}

//nolint:gochecknoglobals
var mapBodies = map[string]mapParameterMasks{
	"charset_map": {
		defaultMasks: ngxConfTake1,
	},
	"geo": {
		specialParameterMasks: map[string]uint{"ranges": ngxConfNoArgs, "proxy_recursive": ngxConfNoArgs},
		defaultMasks:          ngxConfTake1,
	},
	"map": {
		specialParameterMasks: map[string]uint{"volatile": ngxConfNoArgs, "hostnames": ngxConfNoArgs},
		defaultMasks:          ngxConfTake1,
	},
	"match": {
		defaultMasks: ngxConf1More,
	},
	"types": {
		defaultMasks: ngxConf1More,
	},
	"split_clients": {
		defaultMasks: ngxConfTake1,
	},
	"geoip2": {
		defaultMasks: ngxConf1More,
	},
	"otel_exporter": {
		defaultMasks: ngxConfTake1,
	},
}

// analyzeMapBody validates the body of a map-like directive. Map-like directives are block directives
// that don't contain nginx directives, and therefore cannot be analyzed in the same way as other blocks.
func analyzeMapBody(fname string, parameter *Directive, term string, mapCtx string) error {
	masks, known := mapBodies[mapCtx]
	// if we're not inside a known map-like directive, don't bother analyzing
	if !known {
		return nil
	}
	if term != ";" {
		return &ParseError{
			What:      fmt.Sprintf(`unexpected "%s"`, term),
			File:      &fname,
			Line:      &parameter.Line,
			Statement: parameter.String(),
			BlockCtx:  mapCtx,
		}
	}

	if mask, ok := masks.specialParameterMasks[parameter.Directive]; ok {
		// use mask to check the parameter's arguments
		if hasValidArguments(mask, parameter.Args) {
			return nil
		}

		return &ParseError{
			What:      "invalid number of parameters",
			File:      &fname,
			Line:      &parameter.Line,
			Statement: parameter.String(),
			BlockCtx:  mapCtx,
		}
	}

	mask := masks.defaultMasks

	// use mask to check the parameter's arguments
	if hasValidArguments(mask, parameter.Args) {
		return nil
	}

	return &ParseError{
		What:      "invalid number of parameters",
		File:      &fname,
		Line:      &parameter.Line,
		Statement: parameter.String(),
		BlockCtx:  mapCtx,
	}
}

func hasValidArguments(mask uint, args []string) bool {
	return ((mask>>len(args)&1) != 0 && len(args) <= 7) || // NOARGS to TAKE7
		((mask&ngxConfFlag) != 0 && len(args) == 1 && validFlag(args[0])) ||
		((mask & ngxConfAny) != 0) ||
		((mask&ngxConf1More) != 0 && len(args) >= 1) ||
		((mask&ngxConf2More) != 0 && len(args) >= 2)
}
