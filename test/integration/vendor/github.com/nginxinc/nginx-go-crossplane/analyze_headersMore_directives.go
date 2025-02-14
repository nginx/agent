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
var moduleHeadersMoreDirectives = map[string][]uint{
	"more_clear_headers": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxHTTPLifConf | ngxConf1More,
	},
	"more_clear_input_headers": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxHTTPLifConf | ngxConf1More,
	},
	"more_set_headers": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxHTTPLifConf | ngxConf1More,
	},
	"more_set_input_headers": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxHTTPLifConf | ngxConf1More,
	},
}

func HeadersMoreDirectivesMatchFn(directive string) ([]uint, bool) {
	masks, matched := moduleHeadersMoreDirectives[directive]
	return masks, matched
}
