/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package crossplane

// Upgrade for .gen.go files. If you don't have access to some private modules,
// please use -skip options to skip them. e.g. go generate -skip="nap".

// Update for headersmore
//go:generate sh -c "sh ./scripts/generate/generate.sh --url https://github.com/openresty/headers-more-nginx-module.git --config-path ./scripts/generate/configs/headersmore_config.json > ./analyze_headersMore_directives.gen.go"

// Update for njs
//go:generate sh -c "sh ./scripts/generate/generate.sh --url https://github.com/nginx/njs.git --config-path ./scripts/generate/configs/njs_config.json > ./analyze_njs_directives.gen.go"

// Update for OSS, filter in config is the directives not in https://nginx.org/en/docs/dirindex.html but in source code.
// Override in config is for the "if" directive. We create a bitmask ngxConfExpr for it in crossplane, which is not in source code.
//go:generate sh -c "sh ./scripts/generate/generate.sh --url https://github.com/nginx/nginx.git --config-path ./scripts/generate/configs/oss_latest_config.json > ./analyze_oss_latest_directives.gen.go"
//go:generate sh -c "sh ./scripts/generate/generate.sh --url https://github.com/nginx/nginx.git --config-path ./scripts/generate/configs/oss_126_config.json --branch branches/stable-1.26 > ./analyze_oss_126_directives.gen.go"
//go:generate sh -c "sh ./scripts/generate/generate.sh --url https://github.com/nginx/nginx.git --config-path ./scripts/generate/configs/oss_124_config.json --branch branches/stable-1.24 > ./analyze_oss_124_directives.gen.go"

// Update for lua, override is for the lua block directives, see https://github.com/nginxinc/nginx-go-crossplane/pull/86.
//go:generate sh -c "sh ./scripts/generate/generate.sh --url https://github.com/openresty/lua-nginx-module.git --config-path ./scripts/generate/configs/lua_config.json  --path ./src > ./analyze_lua_directives.gen.go"

// Update for otel. Filter is for some directives withou context.
// Otel provides its own config handler for some directives and they don't have context. Currently we don't support them.
//go:generate sh -c "sh ./scripts/generate/generate.sh --url https://github.com/nginxinc/nginx-otel.git --config-path ./scripts/generate/configs/otel_config.json --branch main > ./analyze_otel_directives.gen.go"

// Update for NAP v4 and v5.
// NAP is a private module. Please ensure you have correct access and put the url.
// and branch of it in environment variable NAP_URL, NAP_V4_BRANCH, and NAP_V5_BRANCH.
// Override is for flag dirctives. NAP used ngxConfTake1 for flag directives, we change them to ngxConfFlag in crossplane.
// NAP v4
//go:generate sh -c "sh ./scripts/generate/generate.sh --url $NAP_URL --config-path ./scripts/generate/configs/nap_v4_config.json --branch $NAP_V4_BRANCH --path ./src > analyze_appProtectWAFv4_directives.gen.go"
// NAP v5
//go:generate sh -c "sh ./scripts/generate/generate.sh --url $NAP_URL --config-path ./scripts/generate/configs/nap_v5_config.json --branch $NAP_V5_BRANCH --path ./src > analyze_appProtectWAFv5_directives.gen.go"

// Update for geoip2
//go:generate sh -c "sh ./scripts/generate/generate.sh --url https://github.com/leev/ngx_http_geoip2_module.git --config-path ./scripts/generate/configs/geoip2_config.json > ./analyze_geoip2_directives.gen.go"
import (
	"fmt"
)

// bit masks for different directive argument styles.
const (
	ngxConfNoArgs = 0x00000001 // 0 args
	ngxConfTake1  = 0x00000002 // 1 args
	ngxConfTake2  = 0x00000004 // 2 args
	ngxConfTake3  = 0x00000008 // 3 args
	ngxConfTake4  = 0x00000010 // 4 args
	ngxConfTake5  = 0x00000020 // 5 args
	ngxConfTake6  = 0x00000040 // 6 args
	// ngxConfTake7  = 0x00000080 // 7 args (currently unused).
	ngxConfBlock = 0x00000100 // followed by block
	ngxConfExpr  = 0x00000200 // directive followed by expression in parentheses `()`
	ngxConfFlag  = 0x00000400 // 'on' or 'off'
	ngxConfAny   = 0x00000800 // >=0 args
	ngxConf1More = 0x00001000 // >=1 args
	ngxConf2More = 0x00002000 // >=2 args

	// some helpful argument style aliases.
	ngxConfTake12   = ngxConfTake1 | ngxConfTake2
	ngxConfTake13   = ngxConfTake1 | ngxConfTake3
	ngxConfTake23   = ngxConfTake2 | ngxConfTake3
	ngxConfTake34   = ngxConfTake3 | ngxConfTake4
	ngxConfTake123  = ngxConfTake12 | ngxConfTake3
	ngxConfTake1234 = ngxConfTake123 | ngxConfTake4

	// bit masks for different directive locations.
	ngxDirectConf     = 0x00010000 // main file (not used)
	ngxMgmtMainConf   = 0x00020000 // mgmt // unique bitmask that may not match NGINX source
	ngxMainConf       = 0x00040000 // main context
	ngxEventConf      = 0x00080000 // events
	ngxMailMainConf   = 0x00100000 // mail
	ngxMailSrvConf    = 0x00200000 // mail > server
	ngxStreamMainConf = 0x00400000 // stream
	ngxStreamSrvConf  = 0x00800000 // stream > server
	ngxStreamUpsConf  = 0x01000000 // stream > upstream
	ngxHTTPMainConf   = 0x02000000 // http
	ngxHTTPSrvConf    = 0x04000000 // http > server
	ngxHTTPLocConf    = 0x08000000 // http > location
	ngxHTTPUpsConf    = 0x10000000 // http > upstream
	ngxHTTPSifConf    = 0x20000000 // http > server > if
	ngxHTTPLifConf    = 0x40000000 // http > location > if
	ngxHTTPLmtConf    = 0x80000000 // http > location > limit_except
)

// helpful directive location alias describing "any" context
// doesn't include ngxHTTPSifConf, ngxHTTPLifConf, ngxHTTPLmtConf, or ngxMgmtMainConf.
const ngxAnyConf = ngxMainConf | ngxEventConf | ngxMailMainConf | ngxMailSrvConf |
	ngxStreamMainConf | ngxStreamSrvConf | ngxStreamUpsConf |
	ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxHTTPUpsConf |
	ngxHTTPSifConf | ngxHTTPLifConf | ngxHTTPLmtConf

// map for getting bitmasks from certain context tuples
//
//nolint:gochecknoglobals
var contexts = map[string]uint{
	blockCtx{}.key():                                   ngxMainConf,
	blockCtx{"events"}.key():                           ngxEventConf,
	blockCtx{"mail"}.key():                             ngxMailMainConf,
	blockCtx{"mail", "server"}.key():                   ngxMailSrvConf,
	blockCtx{"stream"}.key():                           ngxStreamMainConf,
	blockCtx{"stream", "server"}.key():                 ngxStreamSrvConf,
	blockCtx{"stream", "upstream"}.key():               ngxStreamUpsConf,
	blockCtx{"http"}.key():                             ngxHTTPMainConf,
	blockCtx{"http", "server"}.key():                   ngxHTTPSrvConf,
	blockCtx{"http", "location"}.key():                 ngxHTTPLocConf,
	blockCtx{"http", "upstream"}.key():                 ngxHTTPUpsConf,
	blockCtx{"http", "server", "if"}.key():             ngxHTTPSifConf,
	blockCtx{"http", "location", "if"}.key():           ngxHTTPLifConf,
	blockCtx{"http", "location", "limit_except"}.key(): ngxHTTPLmtConf,
	blockCtx{"mgmt"}.key():                             ngxMgmtMainConf,
}

func enterBlockCtx(stmt *Directive, ctx blockCtx) blockCtx {
	// don't nest because ngxHTTPLocConf just means "location block in http"
	if len(ctx) > 0 && ctx[0] == "http" && stmt.Directive == "location" {
		return blockCtx{"http", "location"}
	}
	// no other block contexts can be nested like location so just append it
	return append(ctx, stmt.Directive)
}

//nolint:gocyclo,funlen,gocognit
func analyze(fname string, stmt *Directive, term string, ctx blockCtx, options *ParseOptions) error {
	var masks []uint
	knownDirective := false

	currCtx, knownContext := contexts[ctx.key()]
	directiveName := stmt.Directive

	// Find all bitmasks from the sources invoker provides.
	for _, matchFn := range options.DirectiveSources {
		if masksInFn, found := matchFn(directiveName); found {
			masks = append(masks, masksInFn...)
			knownDirective = true
		}
	}

	// If DirectiveSources was not provided, DefaultDirectivesMatchFunc will be used
	// for validation
	if len(options.DirectiveSources) == 0 {
		masks, knownDirective = DefaultDirectivesMatchFunc(directiveName)
	}

	// if strict and directive isn't recognized then throw error
	if options.ErrorOnUnknownDirectives && !knownDirective {
		return &ParseError{
			What:      fmt.Sprintf(`unknown directive "%s"`, stmt.Directive),
			File:      &fname,
			Line:      &stmt.Line,
			Statement: stmt.String(),
			BlockCtx:  ctx.getLastBlock(),
		}
	}

	// if we don't know where this directive is allowed and how
	// many arguments it can take then don't bother analyzing it
	if !knownContext || !knownDirective {
		return nil
	}

	// if this directive can't be used in this context then throw an error
	var ctxMasks []uint
	if options.SkipDirectiveContextCheck {
		ctxMasks = masks
	} else {
		for _, mask := range masks {
			if (mask & currCtx) != 0 {
				ctxMasks = append(ctxMasks, mask)
			}
		}
		if len(ctxMasks) == 0 && !options.SkipDirectiveContextCheck {
			return &ParseError{
				What:      fmt.Sprintf(`"%s" directive is not allowed here`, stmt.Directive),
				File:      &fname,
				Line:      &stmt.Line,
				Statement: stmt.String(),
				BlockCtx:  ctx.getLastBlock(),
			}
		}
	}

	if options.SkipDirectiveArgsCheck {
		return nil
	}

	// do this in reverse because we only throw errors at the end if no masks
	// are valid, and typically the first bit mask is what the parser expects
	var what string
	for i := 0; i < len(ctxMasks); i++ {
		mask := ctxMasks[i]
		// if the directive is an expression type, there must be '(' 'expr' ')' args
		if (mask&ngxConfExpr) > 0 && !validExpr(stmt) {
			what = fmt.Sprintf(`directive "%s"'s is not enclosed in parentheses`, stmt.Directive)
			continue
		}

		// if the directive isn't a block but should be according to the mask
		if (mask&ngxConfBlock) != 0 && term != "{" {
			what = fmt.Sprintf(`directive "%s" has no opening "{"`, stmt.Directive)
			continue
		}

		// if the directive is a block but shouldn't be according to the mask
		if (mask&ngxConfBlock) == 0 && term != ";" {
			what = fmt.Sprintf(`directive "%s" is not terminated by ";"`, stmt.Directive)
			continue
		}

		// use mask to check the directive's arguments
		//nolint:gocritic
		if ((mask>>len(stmt.Args)&1) != 0 && len(stmt.Args) <= 7) || // NOARGS to TAKE7
			((mask&ngxConfFlag) != 0 && len(stmt.Args) == 1 && validFlag(stmt.Args[0])) ||
			((mask & ngxConfAny) != 0) ||
			((mask&ngxConf1More) != 0 && len(stmt.Args) >= 1) ||
			((mask&ngxConf2More) != 0 && len(stmt.Args) >= 2) {
			return nil
		} else if (mask&ngxConfFlag) != 0 && len(stmt.Args) == 1 && !validFlag(stmt.Args[0]) {
			what = fmt.Sprintf(`invalid value "%s" in "%s" directive, it must be "on" or "off"`, stmt.Args[0], stmt.Directive)
		} else {
			what = fmt.Sprintf(`invalid number of arguments in "%s" directive`, stmt.Directive)
		}
	}

	return &ParseError{
		What:      what,
		File:      &fname,
		Line:      &stmt.Line,
		Statement: stmt.String(),
		BlockCtx:  ctx.getLastBlock(),
	}
}

func unionBitmaskMaps(maps ...map[string][]uint) map[string][]uint {
	union := make(map[string][]uint)

	for _, m := range maps {
		for key, value := range m {
			union[key] = value
		}
	}

	return union
}

// A default map for directives, used when ParseOptions.DirectiveSources is
// not provided. It is union of latest Nplus, Njs, and Otel.
//
//nolint:gochecknoglobals
var defaultDirectives = unionBitmaskMaps(nginxPlusLatestDirectives, njsDirectives, otelDirectives)

func DefaultDirectivesMatchFunc(directive string) ([]uint, bool) {
	masks, matched := defaultDirectives[directive]
	return masks, matched
}
