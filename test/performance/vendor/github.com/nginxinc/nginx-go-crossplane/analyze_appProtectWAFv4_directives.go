package crossplane

// nginx app protect specific and global directives, inferred from
// [https://docs.nginx.com/nginx-app-protect/configuration-guide/configuration/#directives]

//nolint:gochecknoglobals
var appProtectWAFv4Directives = map[string][]uint{
	"app_protect_compressed_requests_action": {
		ngxHTTPMainConf | ngxConfTake1,
	},
	"app_protect_cookie_seed": {
		ngxHTTPMainConf | ngxConfTake1,
	},
	"app_protect_cpu_thresholds": {
		ngxHTTPMainConf | ngxConfTake2,
	},
	"app_protect_enable": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfFlag,
	},
	"app_protect_failure_mode_action": {
		ngxHTTPMainConf | ngxConfTake1,
	},
	"app_protect_physical_memory_util_thresholds": {
		ngxHTTPMainConf | ngxConfTake2,
	},
	"app_protect_policy_file": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake1,
	},
	"app_protect_reconnect_period_seconds": {
		ngxHTTPMainConf | ngxConfTake1,
	},
	"app_protect_request_buffer_overflow_action": {
		ngxHTTPMainConf | ngxConfTake1,
	},
	"app_protect_security_log_enable": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfFlag,
	},
	"app_protect_security_log": {
		ngxHTTPMainConf | ngxHTTPSrvConf | ngxHTTPLocConf | ngxConfTake2,
	},
	"app_protect_user_defined_signatures": {
		ngxHTTPMainConf | ngxConfTake1,
	},
}

// AppProtectWAFv4DirectivesMatchFn is a match function for parsing an NGINX config that contains the
// App Protect v4 module.
func AppProtectWAFv4DirectivesMatchFn(directive string) ([]uint, bool) {
	masks, matched := appProtectWAFv4Directives[directive]
	return masks, matched
}
