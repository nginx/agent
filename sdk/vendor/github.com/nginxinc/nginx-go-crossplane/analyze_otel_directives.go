/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

// All the definitions are extracted from the source code
// Each bit mask describes these behaviors:
//   - how many arguments the directive can take
//   - whether or not it is a block directive
//   - whether this is a flag (takes one argument that's either "on" or "off")
//   - which contexts it's allowed to be in

package crossplane

//nolint:gochecknoglobals
var moduleOtelDirectives = map[string][]uint{
	"batch_count": {
		ngxConfTake1,
	},
	"batch_size": {
		ngxConfTake1,
	},
	"endpoint": {
		ngxConfTake1,
	},
	"interval": {
		ngxConfTake1,
	},
	"otel_exporter": {
		ngxHTTPMainConf | ngxConfBlock | ngxConfNoArgs,
	},
	"otel_service_name": {
		ngxHTTPMainConf | ngxConfTake1,
	},
	"otel_span_attr": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake2,
	},
	"otel_span_name": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake1,
	},
	"otel_trace": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake1,
	},
	"otel_trace_context": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake1,
	},
}

func OtelDirectivesMatchFn(directive string) ([]uint, bool) {
	masks, matched := moduleOtelDirectives[directive]
	return masks, matched
}
