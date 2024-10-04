package events

const (
	// Types
	NGINX_EVENT_TYPE = "Nginx"
	AGENT_EVENT_TYPE = "Agent"

	// Categories
	STATUS_CATEGORY      = "Status"
	CONFIG_CATEGORY      = "Config"
	APP_PROTECT_CATEGORY = "AppProtect"

	// Event Levels
	INFO_EVENT_LEVEL     = "INFO"
	DEBUG_EVENT_LEVEL    = "DEBUG"
	WARN_EVENT_LEVEL     = "WARN"
	ERROR_EVENT_LEVEL    = "ERROR"
	CRITICAL_EVENT_LEVEL = "CRITICAL"

	// Messages
	AGENT_START_MESSAGE = "nginx-agent %s started on %s with pid %s"
	AGENT_STOP_MESSAGE  = "nginx-agent %s (pid: %s) stopped on %s"
)
