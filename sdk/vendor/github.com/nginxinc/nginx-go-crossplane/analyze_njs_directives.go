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
var moduleNjsDirectives = map[string][]uint{
	"js_access": {
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake1,
	},
	"js_body_filter": {
		ngxHTTPLocConf | ngxHTTPLifConf | ngxHTTPLmtConf | ngxConfTake12,
	},
	"js_content": {
		ngxHTTPLocConf | ngxHTTPLifConf | ngxHTTPLmtConf | ngxConfTake1,
	},
	"js_fetch_buffer_size": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake1,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake1,
	},
	"js_fetch_ciphers": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake1,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake1,
	},
	"js_fetch_max_response_buffer_size": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake1,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake1,
	},
	"js_fetch_protocols": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConf1More,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConf1More,
	},
	"js_fetch_timeout": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake1,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake1,
	},
	"js_fetch_trusted_certificate": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake1,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake1,
	},
	"js_fetch_verify": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfFlag,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfFlag,
	},
	"js_fetch_verify_depth": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake1,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake1,
	},
	"js_filter": {
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake1,
	},
	"js_header_filter": {
		ngxHTTPLocConf | ngxHTTPLifConf | ngxHTTPLmtConf | ngxConfTake1,
	},
	"js_import": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake13,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake13,
	},
	"js_path": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake1,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake1,
	},
	"js_periodic": {
		ngxHTTPLocConf | ngxConfAny,
		ngxStreamSrvConf | ngxConfAny,
	},
	"js_preload_object": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake13,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake13,
	},
	"js_preread": {
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake1,
	},
	"js_set": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake2,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake2,
	},
	"js_shared_dict_zone": {
		ngxHTTPMainConf | ngxConf1More,
		ngxStreamMainConf | ngxConf1More,
	},
	"js_var": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake12,
		ngxStreamMainConf | ngxStreamSrvConf | ngxConfTake12,
	},
}

func NjsDirectivesMatchFn(directive string) ([]uint, bool) {
	masks, matched := moduleNjsDirectives[directive]
	return masks, matched
}
