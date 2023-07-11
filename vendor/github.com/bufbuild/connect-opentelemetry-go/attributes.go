// Copyright 2022-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelconnect

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/bufbuild/connect-go"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// AttributeFilter is used to filter attributes out based on the [Request] and [attribute.KeyValue].
// If the filter returns true the attribute will be kept else it will be removed.
// AttributeFilter must be safe to call concurrently.
type AttributeFilter func(*Request, attribute.KeyValue) bool

func (filter AttributeFilter) filter(request *Request, values ...attribute.KeyValue) []attribute.KeyValue {
	if filter == nil {
		return values
	}
	// Assign a new slice of zero length with the same underlying
	// array as the values slice. This avoids unnecessary memory allocations.
	filteredValues := values[:0]
	for _, attr := range values {
		if filter(request, attr) {
			filteredValues = append(filteredValues, attr)
		}
	}
	for i := len(filteredValues); i < len(values); i++ {
		values[i] = attribute.KeyValue{}
	}
	return filteredValues
}

func procedureAttributes(procedure string) []attribute.KeyValue {
	parts := strings.SplitN(procedure, "/", 2)
	var attrs []attribute.KeyValue
	switch len(parts) {
	case 0:
		return attrs // invalid
	case 1:
		// fall back to treating the whole string as the method
		if method := parts[0]; method != "" {
			attrs = append(attrs, semconv.RPCMethodKey.String(method))
		}
	default:
		if svc := parts[0]; svc != "" {
			attrs = append(attrs, semconv.RPCServiceKey.String(svc))
		}
		if method := parts[1]; method != "" {
			attrs = append(attrs, semconv.RPCMethodKey.String(method))
		}
	}
	return attrs
}

func requestAttributes(req *Request) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	if addr := req.Peer.Addr; addr != "" {
		attrs = append(attrs, addressAttributes(addr)...)
	}
	name := strings.TrimLeft(req.Spec.Procedure, "/")
	protocol := protocolToSemConv(req.Peer.Protocol)
	attrs = append(attrs, semconv.RPCSystemKey.String(protocol))
	attrs = append(attrs, procedureAttributes(name)...)
	return attrs
}

func addressAttributes(address string) []attribute.KeyValue {
	if host, port, err := net.SplitHostPort(address); err == nil {
		portInt, err := strconv.Atoi(port)
		if err == nil {
			return []attribute.KeyValue{
				semconv.NetPeerNameKey.String(host),
				semconv.NetPeerPortKey.Int(portInt),
			}
		}
	}
	return []attribute.KeyValue{semconv.NetPeerNameKey.String(address)}
}

func statusCodeAttribute(protocol string, serverErr error) (attribute.KeyValue, bool) {
	// Following the respective specifications, use integers and "status_code" for
	// gRPC codes in contrast to strings and "error_code" for Connect codes.
	switch protocol {
	case grpcProtocol, grpcwebProtocol:
		codeKey := attribute.Key("rpc." + protocol + ".status_code")
		if serverErr != nil {
			return codeKey.Int64(int64(connect.CodeOf(serverErr))), true
		}
		return codeKey.Int64(0), true // gRPC uses 0 for success
	case connectProtocol:
		codeKey := attribute.Key("rpc." + protocol + ".error_code")
		if serverErr != nil {
			return codeKey.String(connect.CodeOf(serverErr).String()), true
		}
	}
	return attribute.KeyValue{}, false
}

func headerAttributes(protocol, eventType string, metadata http.Header, allowedKeys []string) []attribute.KeyValue {
	attributes := make([]attribute.KeyValue, 0, len(allowedKeys))
	for _, allowedKey := range allowedKeys {
		if val, ok := metadata[allowedKey]; ok {
			keyValue := attribute.StringSlice(
				formatHeaderAttributeKey(protocol, eventType, allowedKey),
				val,
			)
			attributes = append(attributes, keyValue)
		}
	}
	return attributes
}

// formatHeaderAttributeKey formats header attributes as suggested by the OpenTelemetry specification:
// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/rpc.md#grpc-request-and-response-metadata
func formatHeaderAttributeKey(protocol, eventType, key string) string {
	key = strings.ReplaceAll(strings.ToLower(key), "-", "_")
	return fmt.Sprintf("rpc.%s.%s.metadata.%s", protocol, eventType, key)
}
