package metric_gen

type Field struct {
	Name                    string
	Size                    int
	Possibilities           []string
	SimpleTestPossibilities []string

	Type      int
	Placement int
}

var fieldOrder = []Field{
	{
		Name: "http.uri", Size: 14, Type: stringType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"/", "/uri1", "/uri2"},
	},
	{
		Name: "http.response_code", Size: 9, Type: intDimensionType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"1", "2", "3"},
	},
	{
		Name: "http.request_method", Size: 4, Type: setStringType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"GET", "POST", "PUT"},
		Possibilities:           []string{"GET", "PUT", "POST", "DELETE", "PATCH"},
	},
	{
		Name: "http.request.count", Size: 1, Type: hardCodedOne, Placement: allPlacement,
		SimpleTestPossibilities: []string{"1"},
	},
	{
		Name: "bytes_in", Size: 1, Type: valueType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"1", "50", "42", "33"},
	},
	{
		Name: "bytes_out", Size: 1, Type: valueType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"33", "421", "153", "11"},
	},
	{
		Name: "environment", Size: 5, Type: setStringType, Placement: allPlacement,
		SimpleTestPossibilities: []string{"env1", "env2"},
		Possibilities:           []string{"env1", "env2", "env3", "env4", "env5"},
	},
	{
		Name: "app", Size: 5, Type: setStringType, Placement: allPlacement,
		SimpleTestPossibilities: []string{"app1", "app2", "app3"},
		Possibilities:           []string{"app1", "app2", "app3", "app4", "app5"},
	},
	{
		Name: "component", Size: 8, Type: setStringType, Placement: allPlacement,
		SimpleTestPossibilities: []string{"comp1", "comp2", "comp3"},
		Possibilities:           []string{"component1", "component2", "component3", "component4", "component5", "component6", "component7"},
	},
	{
		Name: "country_code", Size: 8, Type: ipType, Placement: allPlacement,
		SimpleTestPossibilities: []string{"0100007fffff00000000000000000000"},
	},
	{
		Name: "http.version_schema", Size: 4, Type: intDimensionType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"2", "1"},
	},
	{
		Name: "http.upstream_addr", Size: 10, Type: stringType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{"backend1.example.com", "backend2.example.com", "backend3.example.com"},
	},
	{
		Name: "upstream_response_code", Size: 9, Type: intDimensionType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{"200", "404", "503"},
	},
	{
		Name: "http.hostname", Size: 14, Type: stringType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"host1", "host2", "host3"},
	},
	{
		Name: "client.network.latency", Size: 1, Type: valueType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"11", "10", "9"},
	},
	{
		Name: "client.ttfb.latency", Size: 1, Type: valueType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"11", "10", "9"},
	},
	{
		Name: "client.request.latency", Size: 1, Type: valueType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"11", "10", "9"},
	},
	{
		Name: "client.response.latency", Size: 1, Type: valueType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"11", "10", "9"},
	},
	{
		Name: "upstream.network.latency", Size: 1, Type: valueType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{"11", "10", "9"},
	},
	{
		Name: "upstream.header.latency", Size: 1, Type: valueType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{"11", "10", "9"},
	},
	{
		Name: "upstream.response.latency", Size: 1, Type: valueType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{},
	},
	{
		Name: "published_api", Size: 8, Type: stringType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{},
	},
	{
		Name: "request_outcome", Size: 3, Type: stringType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"outcome1", "outcome2"},
	},
	{
		Name: "request_outcome_reason", Size: 5, Type: stringType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{},
	},
	{
		Name: "gateway", Size: 5, Type: setStringType, Placement: allPlacement,
		SimpleTestPossibilities: []string{"gw1", "gw2", "gw3"},
		Possibilities:           []string{"gw1", "gw2", "gw3", "gw4", "gw5", "gw6"},
	},
	{
		Name: "waf.signature_ids", Size: 14, Type: stringType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{},
	},
	{
		Name: "waf.attack_types", Size: 3, Type: stringType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{},
	},
	{
		Name: "waf.violation_rating", Size: 3, Type: stringType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{},
	},
	{
		Name: "waf.violations", Size: 7, Type: stringType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{},
	},
	{
		Name: "waf.violation_subviolation", Size: 4, Type: stringType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{},
	},
	{
		Name: "client.latency", Size: 1, Type: valueType, Placement: HTTPPlacement,
		SimpleTestPossibilities: []string{"11", "10", "9"},
	},
	{
		Name: "upstream.latency", Size: 1, Type: valueType, Placement: NONEPlacement,
		SimpleTestPossibilities: []string{},
	},
	{
		Name: "connection_duration", Size: 1, Type: valueType, Placement: TCPPlacement,
		SimpleTestPossibilities: []string{"11", "10", "9"},
	},
	{
		Name: "family", Size: 2, Type: setStringType, Placement: allPlacement,
		SimpleTestPossibilities: []string{"web", "tcp-upd"},
		Possibilities:           []string{"web", "tcp-udp"},
	},
	{
		Name: "proxied_protocol", Size: 2, Type: setStringType, Placement: allPlacement,
		SimpleTestPossibilities: []string{"http", "tcp"},
		Possibilities:           []string{"http", "tcp"},
	},
	{
		Name: "bytes_rcvd", Size: 1, Placement: allPlacement, Type: valueType,
		SimpleTestPossibilities: []string{"110", "13", "95"},
	},
	{
		Name: "bytes_sent", Size: 1, Placement: allPlacement, Type: valueType,
		SimpleTestPossibilities: []string{"110", "13", "95"},
	},
}
