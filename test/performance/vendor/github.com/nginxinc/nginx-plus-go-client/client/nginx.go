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
	"time"
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

// ErrUnsupportedVer means that client's API version is not supported by NGINX plus API.
var ErrUnsupportedVer = errors.New("API version of the client is not supported by running NGINX Plus")

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
	RequestID string `json:"request_id"`
	Href      string
	Error     apiError
}

func (resp *apiErrorResponse) toString() string {
	return fmt.Sprintf("error.status=%v; error.text=%v; error.code=%v; request_id=%v; href=%v",
		resp.Error.Status, resp.Error.Text, resp.Error.Code, resp.RequestID, resp.Href)
}

type apiError struct {
	Text   string
	Code   string
	Status int
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
	Zone       string
	Peers      []Peer
	Queue      Queue
	Keepalives int
	Zombies    int
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
		return nil, errors.New("http client is not set")
	}

	if !versionSupported(c.apiVersion) {
		return nil, fmt.Errorf("API version %v is not supported by the client", c.apiVersion)
	}

	if c.checkAPI {
		versions, err := getAPIVersions(c.httpClient, apiEndpoint)
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
			return nil, fmt.Errorf("API version %v is not supported by the server", c.apiVersion)
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

func getAPIVersions(httpClient *http.Client, endpoint string) (*versions, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
		return nil, fmt.Errorf("%v is not accessible: expected %v response, got %v", endpoint, http.StatusOK, resp.StatusCode)
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
func (client *NginxClient) CheckIfUpstreamExists(upstream string) error {
	_, err := client.GetHTTPServers(upstream)
	return err
}

// GetHTTPServers returns the servers of the upstream from NGINX.
func (client *NginxClient) GetHTTPServers(upstream string) ([]UpstreamServer, error) {
	path := fmt.Sprintf("http/upstreams/%v/servers", upstream)

	var servers []UpstreamServer
	err := client.get(path, &servers)
	if err != nil {
		return nil, fmt.Errorf("failed to get the HTTP servers of upstream %v: %w", upstream, err)
	}

	return servers, nil
}

// AddHTTPServer adds the server to the upstream.
func (client *NginxClient) AddHTTPServer(upstream string, server UpstreamServer) error {
	id, err := client.getIDOfHTTPServer(upstream, server.Server)
	if err != nil {
		return fmt.Errorf("failed to add %v server to %v upstream: %w", server.Server, upstream, err)
	}
	if id != -1 {
		return fmt.Errorf("failed to add %v server to %v upstream: server already exists", server.Server, upstream)
	}

	path := fmt.Sprintf("http/upstreams/%v/servers/", upstream)
	err = client.post(path, &server)
	if err != nil {
		return fmt.Errorf("failed to add %v server to %v upstream: %w", server.Server, upstream, err)
	}

	return nil
}

// DeleteHTTPServer the server from the upstream.
func (client *NginxClient) DeleteHTTPServer(upstream string, server string) error {
	id, err := client.getIDOfHTTPServer(upstream, server)
	if err != nil {
		return fmt.Errorf("failed to remove %v server from  %v upstream: %w", server, upstream, err)
	}
	if id == -1 {
		return fmt.Errorf("failed to remove %v server from %v upstream: server doesn't exist", server, upstream)
	}

	path := fmt.Sprintf("http/upstreams/%v/servers/%v", upstream, id)
	err = client.delete(path, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to remove %v server from %v upstream: %w", server, upstream, err)
	}

	return nil
}

// UpdateHTTPServers updates the servers of the upstream.
// Servers that are in the slice, but don't exist in NGINX will be added to NGINX.
// Servers that aren't in the slice, but exist in NGINX, will be removed from NGINX.
// Servers that are in the slice and exist in NGINX, but have different parameters, will be updated.
func (client *NginxClient) UpdateHTTPServers(upstream string, servers []UpstreamServer) (added []UpstreamServer, deleted []UpstreamServer, updated []UpstreamServer, err error) {
	serversInNginx, err := client.GetHTTPServers(upstream)
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
		err := client.AddHTTPServer(upstream, server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update servers of %v upstream: %w", upstream, err)
		}
	}

	for _, server := range toDelete {
		err := client.DeleteHTTPServer(upstream, server.Server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update servers of %v upstream: %w", upstream, err)
		}
	}

	for _, server := range toUpdate {
		err := client.UpdateHTTPServer(upstream, server)
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

func (client *NginxClient) getIDOfHTTPServer(upstream string, name string) (int, error) {
	servers, err := client.GetHTTPServers(upstream)
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

func (client *NginxClient) get(path string, data interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

func (client *NginxClient) post(path string, input interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

func (client *NginxClient) delete(path string, expectedStatusCode int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

func (client *NginxClient) patch(path string, input interface{}, expectedStatusCode int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
func (client *NginxClient) CheckIfStreamUpstreamExists(upstream string) error {
	_, err := client.GetStreamServers(upstream)
	return err
}

// GetStreamServers returns the stream servers of the upstream from NGINX.
func (client *NginxClient) GetStreamServers(upstream string) ([]StreamUpstreamServer, error) {
	path := fmt.Sprintf("stream/upstreams/%v/servers", upstream)

	var servers []StreamUpstreamServer
	err := client.get(path, &servers)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream servers of upstream server %v: %w", upstream, err)
	}
	return servers, nil
}

// AddStreamServer adds the stream server to the upstream.
func (client *NginxClient) AddStreamServer(upstream string, server StreamUpstreamServer) error {
	id, err := client.getIDOfStreamServer(upstream, server.Server)
	if err != nil {
		return fmt.Errorf("failed to add %v stream server to %v upstream: %w", server.Server, upstream, err)
	}
	if id != -1 {
		return fmt.Errorf("failed to add %v stream server to %v upstream: server already exists", server.Server, upstream)
	}

	path := fmt.Sprintf("stream/upstreams/%v/servers/", upstream)
	err = client.post(path, &server)
	if err != nil {
		return fmt.Errorf("failed to add %v stream server to %v upstream: %w", server.Server, upstream, err)
	}
	return nil
}

// DeleteStreamServer the server from the upstream.
func (client *NginxClient) DeleteStreamServer(upstream string, server string) error {
	id, err := client.getIDOfStreamServer(upstream, server)
	if err != nil {
		return fmt.Errorf("failed to remove %v stream server from  %v upstream: %w", server, upstream, err)
	}
	if id == -1 {
		return fmt.Errorf("failed to remove %v stream server from %v upstream: server doesn't exist", server, upstream)
	}

	path := fmt.Sprintf("stream/upstreams/%v/servers/%v", upstream, id)
	err = client.delete(path, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to remove %v stream server from %v upstream: %w", server, upstream, err)
	}
	return nil
}

// UpdateStreamServers updates the servers of the upstream.
// Servers that are in the slice, but don't exist in NGINX will be added to NGINX.
// Servers that aren't in the slice, but exist in NGINX, will be removed from NGINX.
// Servers that are in the slice and exist in NGINX, but have different parameters, will be updated.
func (client *NginxClient) UpdateStreamServers(upstream string, servers []StreamUpstreamServer) (added []StreamUpstreamServer, deleted []StreamUpstreamServer, updated []StreamUpstreamServer, err error) {
	serversInNginx, err := client.GetStreamServers(upstream)
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
		err := client.AddStreamServer(upstream, server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update stream servers of %v upstream: %w", upstream, err)
		}
	}

	for _, server := range toDelete {
		err := client.DeleteStreamServer(upstream, server.Server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update stream servers of %v upstream: %w", upstream, err)
		}
	}

	for _, server := range toUpdate {
		err := client.UpdateStreamServer(upstream, server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update stream servers of %v upstream: %w", upstream, err)
		}
	}

	return toAdd, toDelete, toUpdate, nil
}

func (client *NginxClient) getIDOfStreamServer(upstream string, name string) (int, error) {
	servers, err := client.GetStreamServers(upstream)
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
func (client *NginxClient) GetStats() (*Stats, error) {
	endpoints, err := client.GetAvailableEndpoints()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	info, err := client.GetNginxInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	caches, err := client.GetCaches()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	processes, err := client.GetProcesses()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	slabs, err := client.GetSlabs()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	cons, err := client.GetConnections()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	requests, err := client.GetHTTPRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	ssl, err := client.GetSSL()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	zones, err := client.GetServerZones()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	upstreams, err := client.GetUpstreams()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	locationZones, err := client.GetLocationZones()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	resolvers, err := client.GetResolvers()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	limitReqs, err := client.GetHTTPLimitReqs()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	limitConnsHTTP, err := client.GetHTTPConnectionsLimit()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	workers, err := client.GetWorkers()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	streamZones := &StreamServerZones{}
	streamUpstreams := &StreamUpstreams{}
	limitConnsStream := &StreamLimitConnections{}
	var streamZoneSync *StreamZoneSync

	if slices.Contains(endpoints, "stream") {
		streamEndpoints, err := client.GetAvailableStreamEndpoints()
		if err != nil {
			return nil, fmt.Errorf("failed to get stats: %w", err)
		}

		if slices.Contains(streamEndpoints, "server_zones") {
			streamZones, err = client.GetStreamServerZones()
			if err != nil {
				return nil, fmt.Errorf("failed to get stats: %w", err)
			}
		}

		if slices.Contains(streamEndpoints, "upstreams") {
			streamUpstreams, err = client.GetStreamUpstreams()
			if err != nil {
				return nil, fmt.Errorf("failed to get stats: %w", err)
			}
		}

		if slices.Contains(streamEndpoints, "limit_conns") {
			limitConnsStream, err = client.GetStreamConnectionsLimit()
			if err != nil {
				return nil, fmt.Errorf("failed to get stats: %w", err)
			}
		}

		if slices.Contains(streamEndpoints, "zone_sync") {
			streamZoneSync, err = client.GetStreamZoneSync()
			if err != nil {
				return nil, fmt.Errorf("failed to get stats: %w", err)
			}
		}
	}

	return &Stats{
		NginxInfo:              *info,
		Caches:                 *caches,
		Processes:              *processes,
		Slabs:                  *slabs,
		Connections:            *cons,
		HTTPRequests:           *requests,
		SSL:                    *ssl,
		ServerZones:            *zones,
		StreamServerZones:      *streamZones,
		Upstreams:              *upstreams,
		StreamUpstreams:        *streamUpstreams,
		StreamZoneSync:         streamZoneSync,
		LocationZones:          *locationZones,
		Resolvers:              *resolvers,
		HTTPLimitRequests:      *limitReqs,
		HTTPLimitConnections:   *limitConnsHTTP,
		StreamLimitConnections: *limitConnsStream,
		Workers:                workers,
	}, nil
}

// GetAvailableEndpoints returns available endpoints in the API.
func (client *NginxClient) GetAvailableEndpoints() ([]string, error) {
	var endpoints []string
	err := client.get("", &endpoints)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints: %w", err)
	}
	return endpoints, nil
}

// GetAvailableStreamEndpoints returns available stream endpoints in the API.
func (client *NginxClient) GetAvailableStreamEndpoints() ([]string, error) {
	var endpoints []string
	err := client.get("stream", &endpoints)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints: %w", err)
	}
	return endpoints, nil
}

// GetNginxInfo returns Nginx stats.
func (client *NginxClient) GetNginxInfo() (*NginxInfo, error) {
	var info NginxInfo
	err := client.get("nginx", &info)
	if err != nil {
		return nil, fmt.Errorf("failed to get info: %w", err)
	}
	return &info, nil
}

// GetCaches returns Cache stats.
func (client *NginxClient) GetCaches() (*Caches, error) {
	var caches Caches
	err := client.get("http/caches", &caches)
	if err != nil {
		return nil, fmt.Errorf("failed to get caches: %w", err)
	}
	return &caches, nil
}

// GetSlabs returns Slabs stats.
func (client *NginxClient) GetSlabs() (*Slabs, error) {
	var slabs Slabs
	err := client.get("slabs", &slabs)
	if err != nil {
		return nil, fmt.Errorf("failed to get slabs: %w", err)
	}
	return &slabs, nil
}

// GetConnections returns Connections stats.
func (client *NginxClient) GetConnections() (*Connections, error) {
	var cons Connections
	err := client.get("connections", &cons)
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}
	return &cons, nil
}

// GetHTTPRequests returns http/requests stats.
func (client *NginxClient) GetHTTPRequests() (*HTTPRequests, error) {
	var requests HTTPRequests
	err := client.get("http/requests", &requests)
	if err != nil {
		return nil, fmt.Errorf("failed to get http requests: %w", err)
	}
	return &requests, nil
}

// GetSSL returns SSL stats.
func (client *NginxClient) GetSSL() (*SSL, error) {
	var ssl SSL
	err := client.get("ssl", &ssl)
	if err != nil {
		return nil, fmt.Errorf("failed to get ssl: %w", err)
	}
	return &ssl, nil
}

// GetServerZones returns http/server_zones stats.
func (client *NginxClient) GetServerZones() (*ServerZones, error) {
	var zones ServerZones
	err := client.get("http/server_zones", &zones)
	if err != nil {
		return nil, fmt.Errorf("failed to get server zones: %w", err)
	}
	return &zones, err
}

// GetStreamServerZones returns stream/server_zones stats.
func (client *NginxClient) GetStreamServerZones() (*StreamServerZones, error) {
	var zones StreamServerZones
	err := client.get("stream/server_zones", &zones)
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

// GetUpstreams returns http/upstreams stats.
func (client *NginxClient) GetUpstreams() (*Upstreams, error) {
	var upstreams Upstreams
	err := client.get("http/upstreams", &upstreams)
	if err != nil {
		return nil, fmt.Errorf("failed to get upstreams: %w", err)
	}
	return &upstreams, nil
}

// GetStreamUpstreams returns stream/upstreams stats.
func (client *NginxClient) GetStreamUpstreams() (*StreamUpstreams, error) {
	var upstreams StreamUpstreams
	err := client.get("stream/upstreams", &upstreams)
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

// GetStreamZoneSync returns stream/zone_sync stats.
func (client *NginxClient) GetStreamZoneSync() (*StreamZoneSync, error) {
	var streamZoneSync StreamZoneSync
	err := client.get("stream/zone_sync", &streamZoneSync)
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

// GetLocationZones returns http/location_zones stats.
func (client *NginxClient) GetLocationZones() (*LocationZones, error) {
	var locationZones LocationZones
	if client.apiVersion < 5 {
		return &locationZones, nil
	}
	err := client.get("http/location_zones", &locationZones)
	if err != nil {
		return nil, fmt.Errorf("failed to get location zones: %w", err)
	}

	return &locationZones, err
}

// GetResolvers returns Resolvers stats.
func (client *NginxClient) GetResolvers() (*Resolvers, error) {
	var resolvers Resolvers
	if client.apiVersion < 5 {
		return &resolvers, nil
	}
	err := client.get("resolvers", &resolvers)
	if err != nil {
		return nil, fmt.Errorf("failed to get resolvers: %w", err)
	}

	return &resolvers, err
}

// GetProcesses returns Processes stats.
func (client *NginxClient) GetProcesses() (*Processes, error) {
	var processes Processes
	err := client.get("processes", &processes)
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
func (client *NginxClient) GetKeyValPairs(zone string) (KeyValPairs, error) {
	return client.getKeyValPairs(zone, httpContext)
}

// GetStreamKeyValPairs fetches key/value pairs for a given Stream zone.
func (client *NginxClient) GetStreamKeyValPairs(zone string) (KeyValPairs, error) {
	return client.getKeyValPairs(zone, streamContext)
}

func (client *NginxClient) getKeyValPairs(zone string, stream bool) (KeyValPairs, error) {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return nil, errors.New("zone required")
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	var keyValPairs KeyValPairs
	err := client.get(path, &keyValPairs)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyvals for %v/%v zone: %w", base, zone, err)
	}
	return keyValPairs, nil
}

// GetAllKeyValPairs fetches all key/value pairs for all HTTP zones.
func (client *NginxClient) GetAllKeyValPairs() (KeyValPairsByZone, error) {
	return client.getAllKeyValPairs(httpContext)
}

// GetAllStreamKeyValPairs fetches all key/value pairs for all Stream zones.
func (client *NginxClient) GetAllStreamKeyValPairs() (KeyValPairsByZone, error) {
	return client.getAllKeyValPairs(streamContext)
}

func (client *NginxClient) getAllKeyValPairs(stream bool) (KeyValPairsByZone, error) {
	base := "http"
	if stream {
		base = "stream"
	}

	path := fmt.Sprintf("%v/keyvals", base)
	var keyValPairsByZone KeyValPairsByZone
	err := client.get(path, &keyValPairsByZone)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyvals for all %v zones: %w", base, err)
	}
	return keyValPairsByZone, nil
}

// AddKeyValPair adds a new key/value pair to a given HTTP zone.
func (client *NginxClient) AddKeyValPair(zone string, key string, val string) error {
	return client.addKeyValPair(zone, key, val, httpContext)
}

// AddStreamKeyValPair adds a new key/value pair to a given Stream zone.
func (client *NginxClient) AddStreamKeyValPair(zone string, key string, val string) error {
	return client.addKeyValPair(zone, key, val, streamContext)
}

func (client *NginxClient) addKeyValPair(zone string, key string, val string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return errors.New("zone required")
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	input := KeyValPairs{key: val}
	err := client.post(path, &input)
	if err != nil {
		return fmt.Errorf("failed to add key value pair for %v/%v zone: %w", base, zone, err)
	}
	return nil
}

// ModifyKeyValPair modifies the value of an existing key in a given HTTP zone.
func (client *NginxClient) ModifyKeyValPair(zone string, key string, val string) error {
	return client.modifyKeyValPair(zone, key, val, httpContext)
}

// ModifyStreamKeyValPair modifies the value of an existing key in a given Stream zone.
func (client *NginxClient) ModifyStreamKeyValPair(zone string, key string, val string) error {
	return client.modifyKeyValPair(zone, key, val, streamContext)
}

func (client *NginxClient) modifyKeyValPair(zone string, key string, val string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return errors.New("zone required")
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	input := KeyValPairs{key: val}
	err := client.patch(path, &input, http.StatusNoContent)
	if err != nil {
		return fmt.Errorf("failed to update key value pair for %v/%v zone: %w", base, zone, err)
	}
	return nil
}

// DeleteKeyValuePair deletes the key/value pair for a key in a given HTTP zone.
func (client *NginxClient) DeleteKeyValuePair(zone string, key string) error {
	return client.deleteKeyValuePair(zone, key, httpContext)
}

// DeleteStreamKeyValuePair deletes the key/value pair for a key in a given Stream zone.
func (client *NginxClient) DeleteStreamKeyValuePair(zone string, key string) error {
	return client.deleteKeyValuePair(zone, key, streamContext)
}

// To delete a key/value pair you set the value to null via the API,
// then NGINX+ will delete the key.
func (client *NginxClient) deleteKeyValuePair(zone string, key string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return errors.New("zone required")
	}

	// map[string]string can't have a nil value so we use a different type here.
	keyval := make(map[string]interface{})
	keyval[key] = nil

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	err := client.patch(path, &keyval, http.StatusNoContent)
	if err != nil {
		return fmt.Errorf("failed to remove key values pair for %v/%v zone: %w", base, zone, err)
	}
	return nil
}

// DeleteKeyValPairs deletes all the key-value pairs in a given HTTP zone.
func (client *NginxClient) DeleteKeyValPairs(zone string) error {
	return client.deleteKeyValPairs(zone, httpContext)
}

// DeleteStreamKeyValPairs deletes all the key-value pairs in a given Stream zone.
func (client *NginxClient) DeleteStreamKeyValPairs(zone string) error {
	return client.deleteKeyValPairs(zone, streamContext)
}

func (client *NginxClient) deleteKeyValPairs(zone string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return errors.New("zone required")
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	err := client.delete(path, http.StatusNoContent)
	if err != nil {
		return fmt.Errorf("failed to remove all key value pairs for %v/%v zone: %w", base, zone, err)
	}
	return nil
}

// UpdateHTTPServer updates the server of the upstream.
func (client *NginxClient) UpdateHTTPServer(upstream string, server UpstreamServer) error {
	path := fmt.Sprintf("http/upstreams/%v/servers/%v", upstream, server.ID)
	server.ID = 0
	err := client.patch(path, &server, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to update %v server to %v upstream: %w", server.Server, upstream, err)
	}

	return nil
}

// UpdateStreamServer updates the stream server of the upstream.
func (client *NginxClient) UpdateStreamServer(upstream string, server StreamUpstreamServer) error {
	path := fmt.Sprintf("stream/upstreams/%v/servers/%v", upstream, server.ID)
	server.ID = 0
	err := client.patch(path, &server, http.StatusOK)
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

// GetHTTPLimitReqs returns http/limit_reqs stats.
func (client *NginxClient) GetHTTPLimitReqs() (*HTTPLimitRequests, error) {
	var limitReqs HTTPLimitRequests
	if client.apiVersion < 6 {
		return &limitReqs, nil
	}
	err := client.get("http/limit_reqs", &limitReqs)
	if err != nil {
		return nil, fmt.Errorf("failed to get http limit requests: %w", err)
	}
	return &limitReqs, nil
}

// GetHTTPConnectionsLimit returns http/limit_conns stats.
func (client *NginxClient) GetHTTPConnectionsLimit() (*HTTPLimitConnections, error) {
	var limitConns HTTPLimitConnections
	if client.apiVersion < 6 {
		return &limitConns, nil
	}
	err := client.get("http/limit_conns", &limitConns)
	if err != nil {
		return nil, fmt.Errorf("failed to get http connections limit: %w", err)
	}
	return &limitConns, nil
}

// GetStreamConnectionsLimit returns stream/limit_conns stats.
func (client *NginxClient) GetStreamConnectionsLimit() (*StreamLimitConnections, error) {
	var limitConns StreamLimitConnections
	if client.apiVersion < 6 {
		return &limitConns, nil
	}
	err := client.get("stream/limit_conns", &limitConns)
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
func (client *NginxClient) GetWorkers() ([]*Workers, error) {
	var workers []*Workers
	if client.apiVersion < 9 {
		return workers, nil
	}
	err := client.get("workers", &workers)
	if err != nil {
		return nil, fmt.Errorf("failed to get workers: %w", err)
	}
	return workers, nil
}
