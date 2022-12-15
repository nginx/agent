package nginx

const (
	NginxConfTemplate = `
master_process on;
worker_processes auto;

error_log /var/log/nginx/error.log warn;

load_module modules/ngx_http_f5_metrics_module.so;
load_module modules/ngx_stream_f5_metrics_module.so;

events {
}


{{ if .HttpBlock }}
{{.HttpBlock.Build}}
{{else}}
http {
	f5_metrics off;
}
{{end}}

{{ if .StreamBlock }}
{{.StreamBlock.Build}}
{{else}}
stream {
	f5_metrics off;
}
{{end}}
`

	HttpBlockTemplate = `
http {
	f5_metrics on;
	f5_metrics_server unix:{{.F5MetricsServer}};

{{ range $key, $value := .F5MetricsMarkers }}
	f5_metrics_marker {{$key}} {{$value}};
{{ end }}

{{ range $name, $upstream := .Upstreams }}
	upstream {{$name}} {
		server {{$upstream.Server}};
	}
{{ end }}

{{ range $server := .Servers }}
	{{$server.Build}}
{{ end }}
}
`

	StreamBlockTemplate = `
stream {
	f5_metrics on;
	f5_metrics_server unix:{{.F5MetricsServer}};

{{ range $key, $value := .F5MetricsMarkers }}
	f5_metrics_marker {{$key}} {{$value}};
{{ end }}

{{ range $name, $upstream := .Upstreams }}
	upstream {{$name}} {
		server {{$upstream.Server}};
	}
{{ end }}

{{ range $server := .Servers }}
	{{$server.Build}}
{{ end }}
}
`

	ServerBlockTemplate = `
	server {
		listen {{.Listen}};

	{{ range $key, $value := .F5MetricsMarkers }}
		f5_metrics_marker {{$key}} {{$value}};
	{{ end }}

	{{ range $key, $value := .Locations }}
		location '{{$key}}' {
			{{$value.Build}}
		}
	{{ end }}
	}
`

	StreamServerBlockTemplate = `
	server {
		listen {{.Listen}};

	{{ range $key, $value := .F5MetricsMarkers }}
		f5_metrics_marker {{$key}} {{$value}};
	{{ end }}
	{{ range $value := .Directives }}
		{{$value}};
	{{ end }}

		proxy_pass {{.UpstreamName}};
	}
`

	LocationBlockTemplate = `
			{{ range $key, $value := .F5MetricsMarkers }}
				f5_metrics_marker {{$key}} {{$value}};
			{{ end }}

			{{ range $value := .Directives }}
				{{$value}};
			{{ end }}

			proxy_pass http://{{.UpstreamName}};
`
)
