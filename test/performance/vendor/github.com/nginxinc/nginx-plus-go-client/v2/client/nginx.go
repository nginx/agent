package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	// APIVersion is the default version of NGINX Plus API supported by the client.
	APIVersion = 9

	pathNotFoundCode  = "PathNotFound"
	streamContext     = true
	httpContext       = false
	defaultServerPort = "80"
)

var (
	supportedAPIVersions = versions{4, 5, 6, 7, 8, 9}

	// Default values for servers in Upstreams.
	defaultMaxConns    = 0
	defaultMaxFails    = 1
	defaultFailTimeout = "10s"
	defaultSlowStart   = "0s"
	defaultBackup      = false
	defaultDown        = false
	defaultWeight      = 1
)

var (
	ErrParameterRequired = errors.New("parameter is required")
	ErrServerNotFound    = errors.New("server not found")
	ErrServerExists      = errors.New("server already exists")
	ErrNotSupported      = errors.New("not supported")
	ErrInvalidTimeout    = errors.New("invalid timeout")
)

// NginxClient lets you access NGINX Plus API.
type NginxClient struct {
	httpClient  *http.Client
	apiEndpoint string
	apiVersion  int
	checkAPI    bool
}

type Option func(*NginxClient)

type versions []int

// UpstreamServer lets you configure HTTP upstreams.
type UpstreamServer struct {
	MaxConns    *int   `json:"max_conns,omitempty"`
	MaxFails    *int   `json:"max_fails,omitempty"`
	Backup      *bool  `json:"backup,omitempty"`
	Down        *bool  `json:"down,omitempty"`
	Weight      *int   `json:"weight,omitempty"`
	Server      string `json:"server"`
	FailTimeout string `json:"fail_timeout,omitempty"`
	SlowStart   string `json:"slow_start,omitempty"`
	Route       string `json:"route,omitempty"`
	Service     string `json:"service,omitempty"`
	ID          int    `json:"id,omitempty"`
	Drain       bool   `json:"drain,omitempty"`
}

// StreamUpstreamServer lets you configure Stream upstreams.
type StreamUpstreamServer struct {
	MaxConns    *int   `json:"max_conns,omitempty"`
	MaxFails    *int   `json:"max_fails,omitempty"`
	Backup      *bool  `json:"backup,omitempty"`
	Down        *bool  `json:"down,omitempty"`
	Weight      *int   `json:"weight,omitempty"`
	Server      string `json:"server"`
	FailTimeout string `json:"fail_timeout,omitempty"`
	SlowStart   string `json:"slow_start,omitempty"`
	Service     string `json:"service,omitempty"`
	ID          int    `json:"id,omitempty"`
}

type apiErrorResponse struct {
	RequestID string   `json:"request_id"`
	Href      string   `json:"href"`
	Error     apiError `json:"error"`
}

func (resp *apiErrorResponse) toString() string {
	return fmt.Sprintf("error.status=%v; error.text=%v; error.code=%v; request_id=%v; href=%v",
		resp.Error.Status, resp.Error.Text, resp.Error.Code, resp.RequestID, resp.Href)
}

type apiError struct {
	Text   string `json:"text"`
	Code   string `json:"code"`
	Status int    `json:"status"`
}

type internalError struct {
	err string
	apiError
}

// Error allows internalError to match the Error interface.
func (internalError *internalError) Error() string {
	return internalError.err
}

// Wrap is a way of including current context while preserving previous error information,
// similar to `return fmt.Errorf("error doing foo, err: %v", err)` but for our internalError type.
func (internalError *internalError) Wrap(err string) *internalError {
	internalError.err = fmt.Sprintf("%v. %v", err, internalError.err)
	return internalError
}

// this is an internal representation of the Stats object including endpoint and streamEndpoint lists.
type extendedStats struct {
	endpoints       []string
	streamEndpoints []string
	Stats
}

func defaultStats() *extendedStats {
	return &extendedStats{
		endpoints:       []string{},
		streamEndpoints: []string{},
		Stats: Stats{
			Upstreams:              map[string]Upstream{},
			ServerZones:            map[string]ServerZone{},
			StreamServerZones:      map[string]StreamServerZone{},
			StreamUpstreams:        map[string]StreamUpstream{},
			Slabs:                  map[string]Slab{},
			Caches:                 map[string]HTTPCache{},
			HTTPLimitConnections:   map[string]LimitConnection{},
			StreamLimitConnections: map[string]LimitConnection{},
			HTTPLimitRequests:      map[string]HTTPLimitRequest{},
			Resolvers:              map[string]Resolver{},
			LocationZones:          map[string]LocationZone{},
			StreamZoneSync:         nil,
			Workers:                []*Workers{},
			NginxInfo:              NginxInfo{},
			SSL:                    SSL{},
			Connections:            Connections{},
			HTTPRequests:           HTTPRequests{},
			Processes:              Processes{},
		},
	}
}

// Stats represents NGINX Plus stats fetched from the NGINX Plus API.
// https://nginx.org/en/docs/http/ngx_http_api_module.html
type Stats struct {
	Upstreams              Upstreams
	ServerZones            ServerZones
	StreamServerZones      StreamServerZones
	StreamUpstreams        StreamUpstreams
	Slabs                  Slabs
	Caches                 Caches
	HTTPLimitConnections   HTTPLimitConnections
	StreamLimitConnections StreamLimitConnections
	HTTPLimitRequests      HTTPLimitRequests
	Resolvers              Resolvers
	LocationZones          LocationZones
	StreamZoneSync         *StreamZoneSync
	Workers                []*Workers
	NginxInfo              NginxInfo
	SSL                    SSL
	Connections            Connections
	HTTPRequests           HTTPRequests
	Processes              Processes
}

// NginxInfo contains general information about NGINX Plus.
type NginxInfo struct {
	Version         string
	Build           string
	Address         string
	LoadTimestamp   string `json:"load_timestamp"`
	Timestamp       string
	Generation      uint64
	ProcessID       uint64 `json:"pid"`
	ParentProcessID uint64 `json:"ppid"`
}

// Caches is a map of cache stats by cache zone.
type Caches = map[string]HTTPCache

// HTTPCache represents a zone's HTTP Cache.
type HTTPCache struct {
	Size        uint64
	MaxSize     uint64 `json:"max_size"`
	Cold        bool
	Hit         CacheStats
	Stale       CacheStats
	Updating    CacheStats
	Revalidated CacheStats
	Miss        CacheStats
	Expired     ExtendedCacheStats
	Bypass      ExtendedCacheStats
}

// CacheStats are basic cache stats.
type CacheStats struct {
	Responses uint64
	Bytes     uint64
}

// ExtendedCacheStats are extended cache stats.
type ExtendedCacheStats struct {
	CacheStats
	ResponsesWritten uint64 `json:"responses_written"`
	BytesWritten     uint64 `json:"bytes_written"`
}

// Connections represents connection related stats.
type Connections struct {
	Accepted uint64
	Dropped  uint64
	Active   uint64
	Idle     uint64
}

// Slabs is map of slab stats by zone name.
type Slabs map[string]Slab

// Slab represents slab related stats.
type Slab struct {
	Slots Slots
	Pages Pages
}

// Pages represents the slab memory usage stats.
type Pages struct {
	Used uint64
	Free uint64
}

// Slots is a map of slots by slot size.
type Slots map[string]Slot

// Slot represents slot related stats.
type Slot struct {
	Used  uint64
	Free  uint64
	Reqs  uint64
	Fails uint64
}

// HTTPRequests represents HTTP request related stats.
type HTTPRequests struct {
	Total   uint64
	Current uint64
}

// SSL represents SSL related stats.
type SSL struct {
	Handshakes       uint64
	HandshakesFailed uint64         `json:"handshakes_failed"`
	SessionReuses    uint64         `json:"session_reuses"`
	NoCommonProtocol uint64         `json:"no_common_protocol"`
	NoCommonCipher   uint64         `json:"no_common_cipher"`
	HandshakeTimeout uint64         `json:"handshake_timeout"`
	PeerRejectedCert uint64         `json:"peer_rejected_cert"`
	VerifyFailures   VerifyFailures `json:"verify_failures"`
}

type VerifyFailures struct {
	NoCert           uint64 `json:"no_cert"`
	ExpiredCert      uint64 `json:"expired_cert"`
	RevokedCert      uint64 `json:"revoked_cert"`
	HostnameMismatch uint64 `json:"hostname_mismatch"`
	Other            uint64 `json:"other"`
}

// ServerZones is map of server zone stats by zone name.
type ServerZones map[string]ServerZone

// ServerZone represents server zone related stats.
type ServerZone struct {
	Processing uint64
	Requests   uint64
	Responses  Responses
	Discarded  uint64
	Received   uint64
	Sent       uint64
	SSL        SSL
}

// StreamServerZones is map of stream server zone stats by zone name.
type StreamServerZones map[string]StreamServerZone

// StreamServerZone represents stream server zone related stats.
type StreamServerZone struct {
	Processing  uint64
	Connections uint64
	Sessions    Sessions
	Discarded   uint64
	Received    uint64
	Sent        uint64
	SSL         SSL
}

// StreamZoneSync represents the sync information per each shared memory zone and the sync information per node in a cluster.
type StreamZoneSync struct {
	Zones  map[string]SyncZone
	Status StreamZoneSyncStatus
}

// SyncZone represents the synchronization status of a shared memory zone.
type SyncZone struct {
	RecordsPending uint64 `json:"records_pending"`
	RecordsTotal   uint64 `json:"records_total"`
}

// StreamZoneSyncStatus represents the status of a shared memory zone.
type StreamZoneSyncStatus struct {
	BytesIn     uint64 `json:"bytes_in"`
	MsgsIn      uint64 `json:"msgs_in"`
	MsgsOut     uint64 `json:"msgs_out"`
	BytesOut    uint64 `json:"bytes_out"`
	NodesOnline uint64 `json:"nodes_online"`
}

// Responses represents HTTP response related stats.
type Responses struct {
	Codes        HTTPCodes
	Responses1xx uint64 `json:"1xx"`
	Responses2xx uint64 `json:"2xx"`
	Responses3xx uint64 `json:"3xx"`
	Responses4xx uint64 `json:"4xx"`
	Responses5xx uint64 `json:"5xx"`
	Total        uint64
}

// HTTPCodes represents HTTP response codes.
type HTTPCodes struct {
	HTTPContinue              uint64 `json:"100,omitempty"`
	HTTPSwitchingProtocols    uint64 `json:"101,omitempty"`
	HTTPProcessing            uint64 `json:"102,omitempty"`
	HTTPOk                    uint64 `json:"200,omitempty"`
	HTTPCreated               uint64 `json:"201,omitempty"`
	HTTPAccepted              uint64 `json:"202,omitempty"`
	HTTPNoContent             uint64 `json:"204,omitempty"`
	HTTPPartialContent        uint64 `json:"206,omitempty"`
	HTTPSpecialResponse       uint64 `json:"300,omitempty"`
	HTTPMovedPermanently      uint64 `json:"301,omitempty"`
	HTTPMovedTemporarily      uint64 `json:"302,omitempty"`
	HTTPSeeOther              uint64 `json:"303,omitempty"`
	HTTPNotModified           uint64 `json:"304,omitempty"`
	HTTPTemporaryRedirect     uint64 `json:"307,omitempty"`
	HTTPBadRequest            uint64 `json:"400,omitempty"`
	HTTPUnauthorized          uint64 `json:"401,omitempty"`
	HTTPForbidden             uint64 `json:"403,omitempty"`
	HTTPNotFound              uint64 `json:"404,omitempty"`
	HTTPNotAllowed            uint64 `json:"405,omitempty"`
	HTTPRequestTimeOut        uint64 `json:"408,omitempty"`
	HTTPConflict              uint64 `json:"409,omitempty"`
	HTTPLengthRequired        uint64 `json:"411,omitempty"`
	HTTPPreconditionFailed    uint64 `json:"412,omitempty"`
	HTTPRequestEntityTooLarge uint64 `json:"413,omitempty"`
	HTTPRequestURITooLarge    uint64 `json:"414,omitempty"`
	HTTPUnsupportedMediaType  uint64 `json:"415,omitempty"`
	HTTPRangeNotSatisfiable   uint64 `json:"416,omitempty"`
	HTTPTooManyRequests       uint64 `json:"429,omitempty"`
	HTTPClose                 uint64 `json:"444,omitempty"`
	HTTPRequestHeaderTooLarge uint64 `json:"494,omitempty"`
	HTTPSCertError            uint64 `json:"495,omitempty"`
	HTTPSNoCert               uint64 `json:"496,omitempty"`
	HTTPToHTTPS               uint64 `json:"497,omitempty"`
	HTTPClientClosedRequest   uint64 `json:"499,omitempty"`
	HTTPInternalServerError   uint64 `json:"500,omitempty"`
	HTTPNotImplemented        uint64 `json:"501,omitempty"`
	HTTPBadGateway            uint64 `json:"502,omitempty"`
	HTTPServiceUnavailable    uint64 `json:"503,omitempty"`
	HTTPGatewayTimeOut        uint64 `json:"504,omitempty"`
	HTTPInsufficientStorage   uint64 `json:"507,omitempty"`
}

// Sessions represents stream session related stats.
type Sessions struct {
	Sessions2xx uint64 `json:"2xx"`
	Sessions4xx uint64 `json:"4xx"`
	Sessions5xx uint64 `json:"5xx"`
	Total       uint64
}

// Upstreams is a map of upstream stats by upstream name.
type Upstreams map[string]Upstream

// Upstream represents upstream related stats.
type Upstream struct {
	Zone      string
	Peers     []Peer
	Queue     Queue
	Keepalive int
	Zombies   int
}

// StreamUpstreams is a map of stream upstream stats by upstream name.
type StreamUpstreams map[string]StreamUpstream

// StreamUpstream represents stream upstream related stats.
type StreamUpstream struct {
	Zone    string
	Peers   []StreamPeer
	Zombies int
}

// Queue represents queue related stats for an upstream.
type Queue struct {
	Size      int
	MaxSize   int `json:"max_size"`
	Overflows uint64
}

// Peer represents peer (upstream server) related stats.
type Peer struct {
	Server       string
	Service      string
	Name         string
	Selected     string
	Downstart    string
	State        string
	Responses    Responses
	SSL          SSL
	HealthChecks HealthChecks `json:"health_checks"`
	Requests     uint64
	ID           int
	MaxConns     int `json:"max_conns"`
	Sent         uint64
	Received     uint64
	Fails        uint64
	Unavail      uint64
	Active       uint64
	Downtime     uint64
	Weight       int
	HeaderTime   uint64 `json:"header_time"`
	ResponseTime uint64 `json:"response_time"`
	Backup       bool
}

// StreamPeer represents peer (stream upstream server) related stats.
type StreamPeer struct {
	Server        string
	Service       string
	Name          string
	Selected      string
	Downstart     string
	State         string
	SSL           SSL
	HealthChecks  HealthChecks `json:"health_checks"`
	Connections   uint64
	Received      uint64
	ID            int
	ConnectTime   int    `json:"connect_time"`
	FirstByteTime int    `json:"first_byte_time"`
	ResponseTime  uint64 `json:"response_time"`
	Sent          uint64
	MaxConns      int `json:"max_conns"`
	Fails         uint64
	Unavail       uint64
	Active        uint64
	Downtime      uint64
	Weight        int
	Backup        bool
}

// HealthChecks represents health check related stats for a peer.
type HealthChecks struct {
	Checks     uint64
	Fails      uint64
	Unhealthy  uint64
	LastPassed bool `json:"last_passed"`
}

// LocationZones represents location_zones related stats.
type LocationZones map[string]LocationZone

// Resolvers represents resolvers related stats.
type Resolvers map[string]Resolver

// LocationZone represents location_zones related stats.
type LocationZone struct {
	Requests  int64
	Responses Responses
	Discarded int64
	Received  int64
	Sent      int64
}

// Resolver represents resolvers related stats.
type Resolver struct {
	Requests  ResolverRequests  `json:"requests"`
	Responses ResolverResponses `json:"responses"`
}

// ResolverRequests represents resolver requests.
type ResolverRequests struct {
	Name int64
	Srv  int64
	Addr int64
}

// ResolverResponses represents resolver responses.
type ResolverResponses struct {
	Noerror  int64
	Formerr  int64
	Servfail int64
	Nxdomain int64
	Notimp   int64
	Refused  int64
	Timedout int64
	Unknown  int64
}

// Processes represents processes related stats.
type Processes struct {
	Respawned int64
}

// HTTPLimitRequest represents HTTP Requests Rate Limiting.
type HTTPLimitRequest struct {
	Passed         uint64
	Delayed        uint64
	Rejected       uint64
	DelayedDryRun  uint64 `json:"delayed_dry_run"`
	RejectedDryRun uint64 `json:"rejected_dry_run"`
}

// HTTPLimitRequests represents limit requests related stats.
type HTTPLimitRequests map[string]HTTPLimitRequest

// LimitConnection represents Connections Limiting.
type LimitConnection struct {
	Passed         uint64
	Rejected       uint64
	RejectedDryRun uint64 `json:"rejected_dry_run"`
}

// HTTPLimitConnections represents limit connections related stats.
type HTTPLimitConnections map[string]LimitConnection

// StreamLimitConnections represents limit connections related stats.
type StreamLimitConnections map[string]LimitConnection

// Workers represents worker connections related stats.
type Workers struct {
	ID          int
	ProcessID   uint64      `json:"pid"`
	HTTP        WorkersHTTP `json:"http"`
	Connections Connections
}

// WorkersHTTP represents HTTP worker connections.
type WorkersHTTP struct {
	HTTPRequests HTTPRequests `json:"requests"`
}

// WithHTTPClient sets the HTTP client to use for accessing the API.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(o *NginxClient) {
		o.httpClient = httpClient
	}
}

// WithAPIVersion sets the API version to use for accessing the API.
func WithAPIVersion(apiVersion int) Option {
	return func(o *NginxClient) {
		o.apiVersion = apiVersion
	}
}

// WithCheckAPI sets the flag to check the API version of the server.
func WithCheckAPI() Option {
	return func(o *NginxClient) {
		o.checkAPI = true
	}
}

// WithMaxAPIVersion sets the API version to the max API version.
func WithMaxAPIVersion() Option {
	return func(o *NginxClient) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		version, err := o.GetMaxAPIVersion(ctx)
		if err != nil {
			return
		}
		o.apiVersion = version
	}
}

// NewNginxClient creates a new NginxClient.
func NewNginxClient(apiEndpoint string, opts ...Option) (*NginxClient, error) {
	c := &NginxClient{
		httpClient:  http.DefaultClient,
		apiEndpoint: apiEndpoint,
		apiVersion:  APIVersion,
		checkAPI:    false,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		return nil, fmt.Errorf("http client: %w", ErrParameterRequired)
	}

	if !versionSupported(c.apiVersion) {
		return nil, fmt.Errorf("API version %v: %w by the client", c.apiVersion, ErrNotSupported)
	}

	if c.checkAPI {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		versions, err := c.getAPIVersions(ctx, c.httpClient, apiEndpoint)
		if err != nil {
			return nil, fmt.Errorf("error accessing the API: %w", err)
		}
		found := false
		for _, v := range *versions {
			if v == c.apiVersion {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("API version %v: %w by the server", c.apiVersion, ErrNotSupported)
		}
	}

	return c, nil
}

func versionSupported(n int) bool {
	for _, version := range supportedAPIVersions {
		if n == version {
			return true
		}
	}
	return false
}

// GetMaxAPIVersion returns the maximum API version supported by the server and the client.
func (client *NginxClient) GetMaxAPIVersion(ctx context.Context) (int, error) {
	serverVersions, err := client.getAPIVersions(ctx, client.httpClient, client.apiEndpoint)
	if err != nil {
		return 0, fmt.Errorf("failed to get max API version: %w", err)
	}

	maxServerVersion := slices.Max(*serverVersions)
	maxClientVersion := slices.Max(supportedAPIVersions)

	if maxServerVersion > maxClientVersion {
		return maxClientVersion, nil
	}

	return maxServerVersion, nil
}

func (client *NginxClient) getAPIVersions(ctx context.Context, httpClient *http.Client, endpoint string) (*versions, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create a get request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%v is not accessible: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, createResponseMismatchError(resp.Body).Wrap(fmt.Sprintf(
			"failed to get endpoint %q, expected %v response, got %v",
			endpoint, http.StatusOK, resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error while reading body of the response: %w", err)
	}

	var vers versions
	err = json.Unmarshal(body, &vers)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling versions, got %q response: %w", string(body), err)
	}

	return &vers, nil
}

func createResponseMismatchError(respBody io.ReadCloser) *internalError {
	apiErrResp, err := readAPIErrorResponse(respBody)
	if err != nil {
		return &internalError{
			err: fmt.Sprintf("failed to read the response body: %v", err),
		}
	}

	return &internalError{
		err:      apiErrResp.toString(),
		apiError: apiErrResp.Error,
	}
}

func readAPIErrorResponse(respBody io.ReadCloser) (*apiErrorResponse, error) {
	body, err := io.ReadAll(respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response body: %w", err)
	}

	var apiErr apiErrorResponse
	err = json.Unmarshal(body, &apiErr)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling apiErrorResponse: got %q response: %w", string(body), err)
	}

	return &apiErr, nil
}

// CheckIfUpstreamExists checks if the upstream exists in NGINX. If the upstream doesn't exist, it returns the error.
func (client *NginxClient) CheckIfUpstreamExists(ctx context.Context, upstream string) error {
	_, err := client.GetHTTPServers(ctx, upstream)
	return err
}

// GetHTTPServers returns the servers of the upstream from NGINX.
func (client *NginxClient) GetHTTPServers(ctx context.Context, upstream string) ([]UpstreamServer, error) {
	path := fmt.Sprintf("http/upstreams/%v/servers", upstream)

	var servers []UpstreamServer
	err := client.get(ctx, path, &servers)
	if err != nil {
		return nil, fmt.Errorf("failed to get the HTTP servers of upstream %v: %w", upstream, err)
	}

	return servers, nil
}

// AddHTTPServer adds the server to the upstream.
func (client *NginxClient) AddHTTPServer(ctx context.Context, upstream string, server UpstreamServer) error {
	id, err := client.getIDOfHTTPServer(ctx, upstream, server.Server)
	if err != nil {
		return fmt.Errorf("failed to add %v server to %v upstream: %w", server.Server, upstream, err)
	}
	if id != -1 {
		return fmt.Errorf("failed to add %v server to %v upstream: %w", server.Server, upstream, ErrServerExists)
	}

	path := fmt.Sprintf("http/upstreams/%v/servers/", upstream)
	err = client.post(ctx, path, &server)
	if err != nil {
		return fmt.Errorf("failed to add %v server to %v upstream: %w", server.Server, upstream, err)
	}

	return nil
}

// DeleteHTTPServer the server from the upstream.
func (client *NginxClient) DeleteHTTPServer(ctx context.Context, upstream string, server string) error {
	id, err := client.getIDOfHTTPServer(ctx, upstream, server)
	if err != nil {
		return fmt.Errorf("failed to remove %v server from  %v upstream: %w", server, upstream, err)
	}
	if id == -1 {
		return fmt.Errorf("failed to remove %v server from %v upstream: %w", server, upstream, ErrServerNotFound)
	}

	path := fmt.Sprintf("http/upstreams/%v/servers/%v", upstream, id)
	err = client.delete(ctx, path, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to remove %v server from %v upstream: %w", server, upstream, err)
	}

	return nil
}

// UpdateHTTPServers updates the servers of the upstream.
// Servers that are in the slice, but don't exist in NGINX will be added to NGINX.
// Servers that aren't in the slice, but exist in NGINX, will be removed from NGINX.
// Servers that are in the slice and exist in NGINX, but have different parameters, will be updated.
func (client *NginxClient) UpdateHTTPServers(ctx context.Context, upstream string, servers []UpstreamServer) (added []UpstreamServer, deleted []UpstreamServer, updated []UpstreamServer, err error) {
	serversInNginx, err := client.GetHTTPServers(ctx, upstream)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to update servers of %v upstream: %w", upstream, err)
	}

	// We assume port 80 if no port is set for servers.
	formattedServers := make([]UpstreamServer, 0, len(servers))
	for _, server := range servers {
		server.Server = addPortToServer(server.Server)
		formattedServers = append(formattedServers, server)
	}

	toAdd, toDelete, toUpdate := determineUpdates(formattedServers, serversInNginx)

	for _, server := range toAdd {
		err := client.AddHTTPServer(ctx, upstream, server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update servers of %v upstream: %w", upstream, err)
		}
	}

	for _, server := range toDelete {
		err := client.DeleteHTTPServer(ctx, upstream, server.Server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update servers of %v upstream: %w", upstream, err)
		}
	}

	for _, server := range toUpdate {
		err := client.UpdateHTTPServer(ctx, upstream, server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update servers of %v upstream: %w", upstream, err)
		}
	}

	return toAdd, toDelete, toUpdate, nil
}

// haveSameParameters checks if a given server has the same parameters as a server already present in NGINX. Order matters.
func haveSameParameters(newServer UpstreamServer, serverNGX UpstreamServer) bool {
	newServer.ID = serverNGX.ID

	if serverNGX.MaxConns != nil && newServer.MaxConns == nil {
		newServer.MaxConns = &defaultMaxConns
	}

	if serverNGX.MaxFails != nil && newServer.MaxFails == nil {
		newServer.MaxFails = &defaultMaxFails
	}

	if serverNGX.FailTimeout != "" && newServer.FailTimeout == "" {
		newServer.FailTimeout = defaultFailTimeout
	}

	if serverNGX.SlowStart != "" && newServer.SlowStart == "" {
		newServer.SlowStart = defaultSlowStart
	}

	if serverNGX.Backup != nil && newServer.Backup == nil {
		newServer.Backup = &defaultBackup
	}

	if serverNGX.Down != nil && newServer.Down == nil {
		newServer.Down = &defaultDown
	}

	if serverNGX.Weight != nil && newServer.Weight == nil {
		newServer.Weight = &defaultWeight
	}

	return reflect.DeepEqual(newServer, serverNGX)
}

func determineUpdates(updatedServers []UpstreamServer, nginxServers []UpstreamServer) (toAdd []UpstreamServer, toRemove []UpstreamServer, toUpdate []UpstreamServer) {
	for _, server := range updatedServers {
		updateFound := false
		for _, serverNGX := range nginxServers {
			if server.Server == serverNGX.Server && !haveSameParameters(server, serverNGX) {
				server.ID = serverNGX.ID
				updateFound = true
				break
			}
		}
		if updateFound {
			toUpdate = append(toUpdate, server)
		}
	}

	for _, server := range updatedServers {
		found := false
		for _, serverNGX := range nginxServers {
			if server.Server == serverNGX.Server {
				found = true
				break
			}
		}
		if !found {
			toAdd = append(toAdd, server)
		}
	}

	for _, serverNGX := range nginxServers {
		found := false
		for _, server := range updatedServers {
			if serverNGX.Server == server.Server {
				found = true
				break
			}
		}
		if !found {
			toRemove = append(toRemove, serverNGX)
		}
	}

	return
}

func (client *NginxClient) getIDOfHTTPServer(ctx context.Context, upstream string, name string) (int, error) {
	servers, err := client.GetHTTPServers(ctx, upstream)
	if err != nil {
		return -1, fmt.Errorf("error getting id of server %v of upstream %v: %w", name, upstream, err)
	}

	for _, s := range servers {
		if s.Server == name {
			return s.ID, nil
		}
	}

	return -1, nil
}

func (client *NginxClient) get(ctx context.Context, path string, data interface{}) error {
	url := fmt.Sprintf("%v/%v/%v", client.apiEndpoint, client.apiVersion, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create a get request: %w", err)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get %v: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return createResponseMismatchError(resp.Body).Wrap(fmt.Sprintf(
			"expected %v response, got %v",
			http.StatusOK, resp.StatusCode))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read the response body: %w", err)
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		return fmt.Errorf("error unmarshaling response %q: %w", string(body), err)
	}
	return nil
}

func (client *NginxClient) post(ctx context.Context, path string, input interface{}) error {
	url := fmt.Sprintf("%v/%v/%v", client.apiEndpoint, client.apiVersion, path)

	jsonInput, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshall input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonInput))
	if err != nil {
		return fmt.Errorf("failed to create a post request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post %v: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return createResponseMismatchError(resp.Body).Wrap(fmt.Sprintf(
			"expected %v response, got %v",
			http.StatusCreated, resp.StatusCode))
	}

	return nil
}

func (client *NginxClient) delete(ctx context.Context, path string, expectedStatusCode int) error {
	path = fmt.Sprintf("%v/%v/%v/", client.apiEndpoint, client.apiVersion, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create a delete request: %w", err)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatusCode {
		return createResponseMismatchError(resp.Body).Wrap(fmt.Sprintf(
			"failed to complete delete request: expected %v response, got %v",
			expectedStatusCode, resp.StatusCode))
	}
	return nil
}

func (client *NginxClient) patch(ctx context.Context, path string, input interface{}, expectedStatusCode int) error {
	path = fmt.Sprintf("%v/%v/%v/", client.apiEndpoint, client.apiVersion, path)

	jsonInput, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshall input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, path, bytes.NewBuffer(jsonInput))
	if err != nil {
		return fmt.Errorf("failed to create a patch request: %w", err)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create patch request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatusCode {
		return createResponseMismatchError(resp.Body).Wrap(fmt.Sprintf(
			"failed to complete patch request: expected %v response, got %v",
			expectedStatusCode, resp.StatusCode))
	}
	return nil
}

// CheckIfStreamUpstreamExists checks if the stream upstream exists in NGINX. If the upstream doesn't exist, it returns the error.
func (client *NginxClient) CheckIfStreamUpstreamExists(ctx context.Context, upstream string) error {
	_, err := client.GetStreamServers(ctx, upstream)
	return err
}

// GetStreamServers returns the stream servers of the upstream from NGINX.
func (client *NginxClient) GetStreamServers(ctx context.Context, upstream string) ([]StreamUpstreamServer, error) {
	path := fmt.Sprintf("stream/upstreams/%v/servers", upstream)

	var servers []StreamUpstreamServer
	err := client.get(ctx, path, &servers)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream servers of upstream server %v: %w", upstream, err)
	}
	return servers, nil
}

// AddStreamServer adds the stream server to the upstream.
func (client *NginxClient) AddStreamServer(ctx context.Context, upstream string, server StreamUpstreamServer) error {
	id, err := client.getIDOfStreamServer(ctx, upstream, server.Server)
	if err != nil {
		return fmt.Errorf("failed to add %v stream server to %v upstream: %w", server.Server, upstream, err)
	}
	if id != -1 {
		return fmt.Errorf("failed to add %v stream server to %v upstream: %w", server.Server, upstream, ErrServerExists)
	}

	path := fmt.Sprintf("stream/upstreams/%v/servers/", upstream)
	err = client.post(ctx, path, &server)
	if err != nil {
		return fmt.Errorf("failed to add %v stream server to %v upstream: %w", server.Server, upstream, err)
	}
	return nil
}

// DeleteStreamServer the server from the upstream.
func (client *NginxClient) DeleteStreamServer(ctx context.Context, upstream string, server string) error {
	id, err := client.getIDOfStreamServer(ctx, upstream, server)
	if err != nil {
		return fmt.Errorf("failed to remove %v stream server from  %v upstream: %w", server, upstream, err)
	}
	if id == -1 {
		return fmt.Errorf("failed to remove %v stream server from %v upstream: %w", server, upstream, ErrServerNotFound)
	}

	path := fmt.Sprintf("stream/upstreams/%v/servers/%v", upstream, id)
	err = client.delete(ctx, path, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to remove %v stream server from %v upstream: %w", server, upstream, err)
	}
	return nil
}

// UpdateStreamServers updates the servers of the upstream.
// Servers that are in the slice, but don't exist in NGINX will be added to NGINX.
// Servers that aren't in the slice, but exist in NGINX, will be removed from NGINX.
// Servers that are in the slice and exist in NGINX, but have different parameters, will be updated.
func (client *NginxClient) UpdateStreamServers(ctx context.Context, upstream string, servers []StreamUpstreamServer) (added []StreamUpstreamServer, deleted []StreamUpstreamServer, updated []StreamUpstreamServer, err error) {
	serversInNginx, err := client.GetStreamServers(ctx, upstream)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to update stream servers of %v upstream: %w", upstream, err)
	}

	formattedServers := make([]StreamUpstreamServer, 0, len(servers))
	for _, server := range servers {
		server.Server = addPortToServer(server.Server)
		formattedServers = append(formattedServers, server)
	}

	toAdd, toDelete, toUpdate := determineStreamUpdates(formattedServers, serversInNginx)

	for _, server := range toAdd {
		err := client.AddStreamServer(ctx, upstream, server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update stream servers of %v upstream: %w", upstream, err)
		}
	}

	for _, server := range toDelete {
		err := client.DeleteStreamServer(ctx, upstream, server.Server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update stream servers of %v upstream: %w", upstream, err)
		}
	}

	for _, server := range toUpdate {
		err := client.UpdateStreamServer(ctx, upstream, server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update stream servers of %v upstream: %w", upstream, err)
		}
	}

	return toAdd, toDelete, toUpdate, nil
}

func (client *NginxClient) getIDOfStreamServer(ctx context.Context, upstream string, name string) (int, error) {
	servers, err := client.GetStreamServers(ctx, upstream)
	if err != nil {
		return -1, fmt.Errorf("error getting id of stream server %v of upstream %v: %w", name, upstream, err)
	}

	for _, s := range servers {
		if s.Server == name {
			return s.ID, nil
		}
	}

	return -1, nil
}

// haveSameParametersForStream checks if a given server has the same parameters as a server already present in NGINX. Order matters.
func haveSameParametersForStream(newServer StreamUpstreamServer, serverNGX StreamUpstreamServer) bool {
	newServer.ID = serverNGX.ID
	if serverNGX.MaxConns != nil && newServer.MaxConns == nil {
		newServer.MaxConns = &defaultMaxConns
	}

	if serverNGX.MaxFails != nil && newServer.MaxFails == nil {
		newServer.MaxFails = &defaultMaxFails
	}

	if serverNGX.FailTimeout != "" && newServer.FailTimeout == "" {
		newServer.FailTimeout = defaultFailTimeout
	}

	if serverNGX.SlowStart != "" && newServer.SlowStart == "" {
		newServer.SlowStart = defaultSlowStart
	}

	if serverNGX.Backup != nil && newServer.Backup == nil {
		newServer.Backup = &defaultBackup
	}

	if serverNGX.Down != nil && newServer.Down == nil {
		newServer.Down = &defaultDown
	}

	if serverNGX.Weight != nil && newServer.Weight == nil {
		newServer.Weight = &defaultWeight
	}

	return reflect.DeepEqual(newServer, serverNGX)
}

func determineStreamUpdates(updatedServers []StreamUpstreamServer, nginxServers []StreamUpstreamServer) (toAdd []StreamUpstreamServer, toRemove []StreamUpstreamServer, toUpdate []StreamUpstreamServer) {
	for _, server := range updatedServers {
		updateFound := false
		for _, serverNGX := range nginxServers {
			if server.Server == serverNGX.Server && !haveSameParametersForStream(server, serverNGX) {
				server.ID = serverNGX.ID
				updateFound = true
				break
			}
		}
		if updateFound {
			toUpdate = append(toUpdate, server)
		}
	}

	for _, server := range updatedServers {
		found := false
		for _, serverNGX := range nginxServers {
			if server.Server == serverNGX.Server {
				found = true
				break
			}
		}
		if !found {
			toAdd = append(toAdd, server)
		}
	}

	for _, serverNGX := range nginxServers {
		found := false
		for _, server := range updatedServers {
			if serverNGX.Server == server.Server {
				found = true
				break
			}
		}
		if !found {
			toRemove = append(toRemove, serverNGX)
		}
	}

	return
}

// GetStats gets process, slab, connection, request, ssl, zone, stream zone, upstream and stream upstream related stats from the NGINX Plus API.
func (client *NginxClient) GetStats(ctx context.Context) (*Stats, error) {
	initialGroup, initialCtx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	stats := defaultStats()
	// Collecting initial stats
	initialGroup.Go(func() error {
		endpoints, err := client.GetAvailableEndpoints(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get available Endpoints: %w", err)
		}

		mu.Lock()
		stats.endpoints = endpoints
		mu.Unlock()
		return nil
	})

	initialGroup.Go(func() error {
		nginxInfo, err := client.GetNginxInfo(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get NGINX info: %w", err)
		}

		mu.Lock()
		stats.NginxInfo = *nginxInfo
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		caches, err := client.GetCaches(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get Caches: %w", err)
		}

		mu.Lock()
		stats.Caches = *caches
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		processes, err := client.GetProcesses(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get Process information: %w", err)
		}

		mu.Lock()
		stats.Processes = *processes
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		slabs, err := client.GetSlabs(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get Slabs: %w", err)
		}

		mu.Lock()
		stats.Slabs = *slabs
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		httpRequests, err := client.GetHTTPRequests(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get HTTP Requests: %w", err)
		}

		mu.Lock()
		stats.HTTPRequests = *httpRequests
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		ssl, err := client.GetSSL(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get SSL: %w", err)
		}

		mu.Lock()
		stats.SSL = *ssl
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		serverZones, err := client.GetServerZones(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get Server Zones: %w", err)
		}

		mu.Lock()
		stats.ServerZones = *serverZones
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		upstreams, err := client.GetUpstreams(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get Upstreams: %w", err)
		}

		mu.Lock()
		stats.Upstreams = *upstreams
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		locationZones, err := client.GetLocationZones(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get Location Zones: %w", err)
		}

		mu.Lock()
		stats.LocationZones = *locationZones
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		resolvers, err := client.GetResolvers(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get Resolvers: %w", err)
		}

		mu.Lock()
		stats.Resolvers = *resolvers
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		httpLimitRequests, err := client.GetHTTPLimitReqs(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get HTTPLimitRequests: %w", err)
		}

		mu.Lock()
		stats.HTTPLimitRequests = *httpLimitRequests
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		httpLimitConnections, err := client.GetHTTPConnectionsLimit(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get HTTPLimitConnections: %w", err)
		}

		mu.Lock()
		stats.HTTPLimitConnections = *httpLimitConnections
		mu.Unlock()

		return nil
	})

	initialGroup.Go(func() error {
		workers, err := client.GetWorkers(initialCtx)
		if err != nil {
			return fmt.Errorf("failed to get Workers: %w", err)
		}

		mu.Lock()
		stats.Workers = workers
		mu.Unlock()

		return nil
	})

	if err := initialGroup.Wait(); err != nil {
		return nil, fmt.Errorf("error returned from contacting Plus API: %w", err)
	}

	// Process stream endpoints if they exist
	if slices.Contains(stats.endpoints, "stream") {
		availableStreamGroup, asgCtx := errgroup.WithContext(ctx)

		availableStreamGroup.Go(func() error {
			streamEndpoints, err := client.GetAvailableStreamEndpoints(asgCtx)
			if err != nil {
				return fmt.Errorf("failed to get available Stream Endpoints: %w", err)
			}

			mu.Lock()
			stats.streamEndpoints = streamEndpoints
			mu.Unlock()

			return nil
		})

		if err := availableStreamGroup.Wait(); err != nil {
			return nil, fmt.Errorf("no useful metrics found in stream stats: %w", err)
		}

		streamGroup, sgCtx := errgroup.WithContext(ctx)

		if slices.Contains(stats.streamEndpoints, "server_zones") {
			streamGroup.Go(func() error {
				streamServerZones, err := client.GetStreamServerZones(sgCtx)
				if err != nil {
					return fmt.Errorf("failed to get streamServerZones: %w", err)
				}

				mu.Lock()
				stats.StreamServerZones = *streamServerZones
				mu.Unlock()

				return nil
			})
		}

		if slices.Contains(stats.streamEndpoints, "upstreams") {
			streamGroup.Go(func() error {
				streamUpstreams, err := client.GetStreamUpstreams(sgCtx)
				if err != nil {
					return fmt.Errorf("failed to get StreamUpstreams: %w", err)
				}

				mu.Lock()
				stats.StreamUpstreams = *streamUpstreams
				mu.Unlock()

				return nil
			})
		}

		if slices.Contains(stats.streamEndpoints, "limit_conns") {
			streamGroup.Go(func() error {
				streamConnectionsLimit, err := client.GetStreamConnectionsLimit(sgCtx)
				if err != nil {
					return fmt.Errorf("failed to get StreamLimitConnections: %w", err)
				}

				mu.Lock()
				stats.StreamLimitConnections = *streamConnectionsLimit
				mu.Unlock()

				return nil
			})

			streamGroup.Go(func() error {
				streamZoneSync, err := client.GetStreamZoneSync(sgCtx)
				if err != nil {
					return fmt.Errorf("failed to get StreamZoneSync: %w", err)
				}

				mu.Lock()
				stats.StreamZoneSync = streamZoneSync
				mu.Unlock()

				return nil
			})
		}

		if err := streamGroup.Wait(); err != nil {
			return nil, fmt.Errorf("no useful metrics found in stream stats: %w", err)
		}
	}

	// Report connection metrics separately so it does not influence the results
	connectionsGroup, cgCtx := errgroup.WithContext(ctx)

	connectionsGroup.Go(func() error {
		// replace this call with a context specific call
		connections, err := client.GetConnections(cgCtx)
		if err != nil {
			return fmt.Errorf("failed to get connections: %w", err)
		}

		mu.Lock()
		stats.Connections = *connections
		mu.Unlock()

		return nil
	})

	if err := connectionsGroup.Wait(); err != nil {
		return nil, fmt.Errorf("connections metrics not found: %w", err)
	}

	return &stats.Stats, nil
}

// GetAvailableEndpoints returns available endpoints in the API.
func (client *NginxClient) GetAvailableEndpoints(ctx context.Context) ([]string, error) {
	var endpoints []string
	err := client.get(ctx, "", &endpoints)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints: %w", err)
	}
	return endpoints, nil
}

// GetAvailableStreamEndpoints returns available stream endpoints in the API with a context.
func (client *NginxClient) GetAvailableStreamEndpoints(ctx context.Context) ([]string, error) {
	var endpoints []string
	err := client.get(ctx, "stream", &endpoints)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints: %w", err)
	}
	return endpoints, nil
}

// GetNginxInfo returns Nginx stats with a context.
func (client *NginxClient) GetNginxInfo(ctx context.Context) (*NginxInfo, error) {
	var info NginxInfo
	err := client.get(ctx, "nginx", &info)
	if err != nil {
		return nil, fmt.Errorf("failed to get info: %w", err)
	}
	return &info, nil
}

// GetCaches returns Cache stats with a context.
func (client *NginxClient) GetCaches(ctx context.Context) (*Caches, error) {
	var caches Caches
	err := client.get(ctx, "http/caches", &caches)
	if err != nil {
		return nil, fmt.Errorf("failed to get caches: %w", err)
	}
	return &caches, nil
}

// GetSlabs returns Slabs stats with a context.
func (client *NginxClient) GetSlabs(ctx context.Context) (*Slabs, error) {
	var slabs Slabs
	err := client.get(ctx, "slabs", &slabs)
	if err != nil {
		return nil, fmt.Errorf("failed to get slabs: %w", err)
	}
	return &slabs, nil
}

// GetConnections returns Connections stats with a context.
func (client *NginxClient) GetConnections(ctx context.Context) (*Connections, error) {
	var cons Connections
	err := client.get(ctx, "connections", &cons)
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}
	return &cons, nil
}

// GetHTTPRequests returns http/requests stats with a context.
func (client *NginxClient) GetHTTPRequests(ctx context.Context) (*HTTPRequests, error) {
	var requests HTTPRequests
	err := client.get(ctx, "http/requests", &requests)
	if err != nil {
		return nil, fmt.Errorf("failed to get http requests: %w", err)
	}
	return &requests, nil
}

// GetSSL returns SSL stats with a context.
func (client *NginxClient) GetSSL(ctx context.Context) (*SSL, error) {
	var ssl SSL
	err := client.get(ctx, "ssl", &ssl)
	if err != nil {
		return nil, fmt.Errorf("failed to get ssl: %w", err)
	}
	return &ssl, nil
}

// GetServerZones returns http/server_zones stats with a context.
func (client *NginxClient) GetServerZones(ctx context.Context) (*ServerZones, error) {
	var zones ServerZones
	err := client.get(ctx, "http/server_zones", &zones)
	if err != nil {
		return nil, fmt.Errorf("failed to get server zones: %w", err)
	}
	return &zones, err
}

// GetStreamServerZones returns stream/server_zones stats with a context.
func (client *NginxClient) GetStreamServerZones(ctx context.Context) (*StreamServerZones, error) {
	var zones StreamServerZones
	err := client.get(ctx, "stream/server_zones", &zones)
	if err != nil {
		var ie *internalError
		if errors.As(err, &ie) {
			if ie.Code == pathNotFoundCode {
				return &zones, nil
			}
		}
		return nil, fmt.Errorf("failed to get stream server zones: %w", err)
	}
	return &zones, err
}

// GetUpstreams returns http/upstreams stats with a context.
func (client *NginxClient) GetUpstreams(ctx context.Context) (*Upstreams, error) {
	var upstreams Upstreams
	err := client.get(ctx, "http/upstreams", &upstreams)
	if err != nil {
		return nil, fmt.Errorf("failed to get upstreams: %w", err)
	}
	return &upstreams, nil
}

// GetStreamUpstreams returns stream/upstreams stats with a context.
func (client *NginxClient) GetStreamUpstreams(ctx context.Context) (*StreamUpstreams, error) {
	var upstreams StreamUpstreams
	err := client.get(ctx, "stream/upstreams", &upstreams)
	if err != nil {
		var ie *internalError
		if errors.As(err, &ie) {
			if ie.Code == pathNotFoundCode {
				return &upstreams, nil
			}
		}
		return nil, fmt.Errorf("failed to get stream upstreams: %w", err)
	}
	return &upstreams, nil
}

// GetStreamZoneSync returns stream/zone_sync stats with a context.
func (client *NginxClient) GetStreamZoneSync(ctx context.Context) (*StreamZoneSync, error) {
	var streamZoneSync StreamZoneSync
	err := client.get(ctx, "stream/zone_sync", &streamZoneSync)
	if err != nil {
		var ie *internalError
		if errors.As(err, &ie) {
			if ie.Code == pathNotFoundCode {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("failed to get stream zone sync: %w", err)
	}

	return &streamZoneSync, err
}

// GetLocationZones returns http/location_zones stats with a context.
func (client *NginxClient) GetLocationZones(ctx context.Context) (*LocationZones, error) {
	var locationZones LocationZones
	if client.apiVersion < 5 {
		return &locationZones, nil
	}
	err := client.get(ctx, "http/location_zones", &locationZones)
	if err != nil {
		return nil, fmt.Errorf("failed to get location zones: %w", err)
	}

	return &locationZones, err
}

// GetResolvers returns Resolvers stats with a context.
func (client *NginxClient) GetResolvers(ctx context.Context) (*Resolvers, error) {
	var resolvers Resolvers
	if client.apiVersion < 5 {
		return &resolvers, nil
	}
	err := client.get(ctx, "resolvers", &resolvers)
	if err != nil {
		return nil, fmt.Errorf("failed to get resolvers: %w", err)
	}

	return &resolvers, err
}

// GetProcesses returns Processes stats with a context.
func (client *NginxClient) GetProcesses(ctx context.Context) (*Processes, error) {
	var processes Processes
	err := client.get(ctx, "processes", &processes)
	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %w", err)
	}

	return &processes, err
}

// KeyValPairs are the key-value pairs stored in a zone.
type KeyValPairs map[string]string

// KeyValPairsByZone are the KeyValPairs for all zones, by zone name.
type KeyValPairsByZone map[string]KeyValPairs

// GetKeyValPairs fetches key/value pairs for a given HTTP zone.
func (client *NginxClient) GetKeyValPairs(ctx context.Context, zone string) (KeyValPairs, error) {
	return client.getKeyValPairs(ctx, zone, httpContext)
}

// GetStreamKeyValPairs fetches key/value pairs for a given Stream zone.
func (client *NginxClient) GetStreamKeyValPairs(ctx context.Context, zone string) (KeyValPairs, error) {
	return client.getKeyValPairs(ctx, zone, streamContext)
}

func (client *NginxClient) getKeyValPairs(ctx context.Context, zone string, stream bool) (KeyValPairs, error) {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return nil, fmt.Errorf("zone: %w", ErrParameterRequired)
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	var keyValPairs KeyValPairs
	err := client.get(ctx, path, &keyValPairs)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyvals for %v/%v zone: %w", base, zone, err)
	}
	return keyValPairs, nil
}

// GetAllKeyValPairs fetches all key/value pairs for all HTTP zones.
func (client *NginxClient) GetAllKeyValPairs(ctx context.Context) (KeyValPairsByZone, error) {
	return client.getAllKeyValPairs(ctx, httpContext)
}

// GetAllStreamKeyValPairs fetches all key/value pairs for all Stream zones.
func (client *NginxClient) GetAllStreamKeyValPairs(ctx context.Context) (KeyValPairsByZone, error) {
	return client.getAllKeyValPairs(ctx, streamContext)
}

func (client *NginxClient) getAllKeyValPairs(ctx context.Context, stream bool) (KeyValPairsByZone, error) {
	base := "http"
	if stream {
		base = "stream"
	}

	path := fmt.Sprintf("%v/keyvals", base)
	var keyValPairsByZone KeyValPairsByZone
	err := client.get(ctx, path, &keyValPairsByZone)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyvals for all %v zones: %w", base, err)
	}
	return keyValPairsByZone, nil
}

// AddKeyValPair adds a new key/value pair to a given HTTP zone.
func (client *NginxClient) AddKeyValPair(ctx context.Context, zone string, key string, val string) error {
	return client.addKeyValPair(ctx, zone, key, val, httpContext)
}

// AddStreamKeyValPair adds a new key/value pair to a given Stream zone.
func (client *NginxClient) AddStreamKeyValPair(ctx context.Context, zone string, key string, val string) error {
	return client.addKeyValPair(ctx, zone, key, val, streamContext)
}

func (client *NginxClient) addKeyValPair(ctx context.Context, zone string, key string, val string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return fmt.Errorf("zone: %w", ErrParameterRequired)
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	input := KeyValPairs{key: val}
	err := client.post(ctx, path, &input)
	if err != nil {
		return fmt.Errorf("failed to add key value pair for %v/%v zone: %w", base, zone, err)
	}
	return nil
}

// ModifyKeyValPair modifies the value of an existing key in a given HTTP zone.
func (client *NginxClient) ModifyKeyValPair(ctx context.Context, zone string, key string, val string) error {
	return client.modifyKeyValPair(ctx, zone, key, val, httpContext)
}

// ModifyStreamKeyValPair modifies the value of an existing key in a given Stream zone.
func (client *NginxClient) ModifyStreamKeyValPair(ctx context.Context, zone string, key string, val string) error {
	return client.modifyKeyValPair(ctx, zone, key, val, streamContext)
}

func (client *NginxClient) modifyKeyValPair(ctx context.Context, zone string, key string, val string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return fmt.Errorf("zone: %w", ErrParameterRequired)
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	input := KeyValPairs{key: val}
	err := client.patch(ctx, path, &input, http.StatusNoContent)
	if err != nil {
		return fmt.Errorf("failed to update key value pair for %v/%v zone: %w", base, zone, err)
	}
	return nil
}

// DeleteKeyValuePair deletes the key/value pair for a key in a given HTTP zone.
func (client *NginxClient) DeleteKeyValuePair(ctx context.Context, zone string, key string) error {
	return client.deleteKeyValuePair(ctx, zone, key, httpContext)
}

// DeleteStreamKeyValuePair deletes the key/value pair for a key in a given Stream zone.
func (client *NginxClient) DeleteStreamKeyValuePair(ctx context.Context, zone string, key string) error {
	return client.deleteKeyValuePair(ctx, zone, key, streamContext)
}

// To delete a key/value pair you set the value to null via the API,
// then NGINX+ will delete the key.
func (client *NginxClient) deleteKeyValuePair(ctx context.Context, zone string, key string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return fmt.Errorf("zone: %w", ErrParameterRequired)
	}

	// map[string]string can't have a nil value so we use a different type here.
	keyval := make(map[string]interface{})
	keyval[key] = nil

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	err := client.patch(ctx, path, &keyval, http.StatusNoContent)
	if err != nil {
		return fmt.Errorf("failed to remove key values pair for %v/%v zone: %w", base, zone, err)
	}
	return nil
}

// DeleteKeyValPairs deletes all the key-value pairs in a given HTTP zone.
func (client *NginxClient) DeleteKeyValPairs(ctx context.Context, zone string) error {
	return client.deleteKeyValPairs(ctx, zone, httpContext)
}

// DeleteStreamKeyValPairs deletes all the key-value pairs in a given Stream zone.
func (client *NginxClient) DeleteStreamKeyValPairs(ctx context.Context, zone string) error {
	return client.deleteKeyValPairs(ctx, zone, streamContext)
}

func (client *NginxClient) deleteKeyValPairs(ctx context.Context, zone string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return fmt.Errorf("zone: %w", ErrParameterRequired)
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	err := client.delete(ctx, path, http.StatusNoContent)
	if err != nil {
		return fmt.Errorf("failed to remove all key value pairs for %v/%v zone: %w", base, zone, err)
	}
	return nil
}

// UpdateHTTPServer updates the server of the upstream.
func (client *NginxClient) UpdateHTTPServer(ctx context.Context, upstream string, server UpstreamServer) error {
	path := fmt.Sprintf("http/upstreams/%v/servers/%v", upstream, server.ID)
	server.ID = 0
	err := client.patch(ctx, path, &server, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to update %v server to %v upstream: %w", server.Server, upstream, err)
	}

	return nil
}

// UpdateStreamServer updates the stream server of the upstream.
func (client *NginxClient) UpdateStreamServer(ctx context.Context, upstream string, server StreamUpstreamServer) error {
	path := fmt.Sprintf("stream/upstreams/%v/servers/%v", upstream, server.ID)
	server.ID = 0
	err := client.patch(ctx, path, &server, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to update %v stream server to %v upstream: %w", server.Server, upstream, err)
	}

	return nil
}

// Version returns client's current N+ API version.
func (client *NginxClient) Version() int {
	return client.apiVersion
}

func addPortToServer(server string) string {
	if len(strings.Split(server, ":")) == 2 {
		return server
	}

	if len(strings.Split(server, "]:")) == 2 {
		return server
	}

	if strings.HasPrefix(server, "unix:") {
		return server
	}

	return fmt.Sprintf("%v:%v", server, defaultServerPort)
}

// GetHTTPLimitReqs returns http/limit_reqs stats with a context.
func (client *NginxClient) GetHTTPLimitReqs(ctx context.Context) (*HTTPLimitRequests, error) {
	var limitReqs HTTPLimitRequests
	if client.apiVersion < 6 {
		return &limitReqs, nil
	}
	err := client.get(ctx, "http/limit_reqs", &limitReqs)
	if err != nil {
		return nil, fmt.Errorf("failed to get http limit requests: %w", err)
	}
	return &limitReqs, nil
}

// GetHTTPConnectionsLimit returns http/limit_conns stats with a context.
func (client *NginxClient) GetHTTPConnectionsLimit(ctx context.Context) (*HTTPLimitConnections, error) {
	var limitConns HTTPLimitConnections
	if client.apiVersion < 6 {
		return &limitConns, nil
	}
	err := client.get(ctx, "http/limit_conns", &limitConns)
	if err != nil {
		return nil, fmt.Errorf("failed to get http connections limit: %w", err)
	}
	return &limitConns, nil
}

// GetStreamConnectionsLimit returns stream/limit_conns stats with a context.
func (client *NginxClient) GetStreamConnectionsLimit(ctx context.Context) (*StreamLimitConnections, error) {
	var limitConns StreamLimitConnections
	if client.apiVersion < 6 {
		return &limitConns, nil
	}
	err := client.get(ctx, "stream/limit_conns", &limitConns)
	if err != nil {
		var ie *internalError
		if errors.As(err, &ie) {
			if ie.Code == pathNotFoundCode {
				return &limitConns, nil
			}
		}
		return nil, fmt.Errorf("failed to get stream connections limit: %w", err)
	}
	return &limitConns, nil
}

// GetWorkers returns workers stats.
func (client *NginxClient) GetWorkers(ctx context.Context) ([]*Workers, error) {
	var workers []*Workers
	if client.apiVersion < 9 {
		return workers, nil
	}
	err := client.get(ctx, "workers", &workers)
	if err != nil {
		return nil, fmt.Errorf("failed to get workers: %w", err)
	}
	return workers, nil
}
