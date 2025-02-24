#include <ngx_config.h>
#include <ngx_core.h>
#include <ngx_http.h>

// Configuration structure for the custom delay module
typedef struct {
    ngx_msec_t delay;  // Delay time in milliseconds
} ngx_http_custom_delay_conf_t;

// Function prototypes
static ngx_int_t ngx_http_custom_delay_handler(ngx_http_request_t *r);
static void *ngx_http_custom_delay_create_conf(ngx_conf_t *cf);
static char *ngx_http_custom_delay_merge_conf(ngx_conf_t *cf, void *parent, void *child);

// Module directives
static ngx_command_t ngx_http_custom_delay_commands[] = {
    { ngx_string("custom_delay"),
      NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_CONF_TAKE1,
      ngx_conf_set_msec_slot,
      NGX_HTTP_LOC_CONF_OFFSET,
      offsetof(ngx_http_custom_delay_conf_t, delay),
      NULL },
    ngx_null_command  // End of commands
};

// Module context
static ngx_http_module_t ngx_http_custom_delay_module_ctx = {
    NULL,                                  // preconfiguration
    NULL,                                  // postconfiguration
    NULL,                                  // create main configuration
    NULL,                                  // init main configuration
    NULL,                                  // create server configuration
    NULL,                                  // merge server configuration
    ngx_http_custom_delay_create_conf,     // create location configuration
    ngx_http_custom_delay_merge_conf       // merge location configuration
};

// Module definition
ngx_module_t ngx_http_custom_delay_module = {
    NGX_MODULE_V1,
    &ngx_http_custom_delay_module_ctx,     // module context
    ngx_http_custom_delay_commands,        // module directives
    NGX_HTTP_MODULE,                       // module type
    NULL,                                  // init master
    NULL,                                  // init module
    NULL,                                  // init process
    NULL,                                  // init thread
    NULL,                                  // exit thread
    NULL,                                  // exit process
    NULL,                                  // exit master
    NGX_MODULE_V1_PADDING
};

// Handler for the delay functionality
static ngx_int_t ngx_http_custom_delay_handler(ngx_http_request_t *r) {
    ngx_http_custom_delay_conf_t *conf;

    // Get the module's configuration
    conf = ngx_http_get_module_loc_conf(r, ngx_http_custom_delay_module);

    // If no delay is set, decline the request
    if (conf->delay == 0) {
        return NGX_DECLINED;
    }

    // Log the delay for debugging
    ngx_log_debug1(NGX_LOG_DEBUG_HTTP, r->connection->log, 0,
                   "custom delay: %M ms", conf->delay);

    // Add a timer to delay the request
    ngx_add_timer(r->connection->write, conf->delay);

    // Return NGX_AGAIN to pause request processing
    return NGX_AGAIN;
}

// Create the module's configuration
static void *ngx_http_custom_delay_create_conf(ngx_conf_t *cf) {
    ngx_http_custom_delay_conf_t *conf;

    // Allocate memory for the configuration
    conf = ngx_pcalloc(cf->pool, sizeof(ngx_http_custom_delay_conf_t));
    if (conf == NULL) {
        return NULL;
    }

    // Initialize the delay to 0 (no delay)
    conf->delay = 0;

    return conf;
}

// Merge configurations from parent and child blocks
static char *ngx_http_custom_delay_merge_conf(ngx_conf_t *cf, void *parent, void *child) {
    ngx_http_custom_delay_conf_t *prev = parent;
    ngx_http_custom_delay_conf_t *conf = child;

    // Merge the delay value
    ngx_conf_merge_msec_value(conf->delay, prev->delay, 0);

    return NGX_CONF_OK;
}

// Initialize the module
static ngx_int_t ngx_http_custom_delay_init(ngx_conf_t *cf) {
    ngx_http_handler_pt *h;
    ngx_http_core_main_conf_t *cmcf;

    // Get the core module's main configuration
    cmcf = ngx_http_conf_get_module_main_conf(cf, ngx_http_core_module);

    // Add the delay handler to the PREACCESS phase
    h = ngx_array_push(&cmcf->phases[NGX_HTTP_PREACCESS_PHASE].handlers);
    if (h == NULL) {
        return NGX_ERROR;
    }

    *h = ngx_http_custom_delay_handler;

    return NGX_OK;
}
