package network

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseToLinuxRouteStruct(t *testing.T) {
	tests := []struct {
		input    []byte
		expected routeStruct
		err      error
	}{
		{
			input: []byte(`
			Iface   Destination     Gateway         Flags   RefCnt  Use     Metric  Mask            MTU     Window  IRTT                                                       
			enp0s3  0002000A        00000000        0001    0       0       0       00FFFFFF        0       0       0                                                                             
			enp0s3  0202000A        00000000        0005    0       0       100     FFFFFFFF        0       0       0                                                                           
			enp0s3  00000000        0202000A        0003    0       0       100     00000000        0       0       0                                                                        
			docker0 000011AC        00000000        0001    0       0       0       0000FFFF        0       0       0                                                                            
			enp0s8  00C0A8C0        00000000        0001    0       0       0       00FFFFFF        0       0       0   `),
			expected: routeStruct{
				Iface:       "enp0s3",
				Destination: "00000000",
				Gateway:     "0202000A",
				Flags:       "0003",
				RefCnt:      "0",
				Use:         "0",
				Metric:      "100",
				Mask:        "00000000",
				MTU:         "0",
				Window:      "0",
				IRTT:        "0",
			},
			err: nil,
		},
		{
			input: []byte(`
			Iface   Destination     Gateway         Flags   RefCnt  Use     Metric  Mask            MTU     Window  IRTT`),
			expected: routeStruct{},
			err:      errors.New("interface with default destination not found"),
		},
		{
			input:    []byte(``),
			expected: routeStruct{},
			err:      errors.New("Invalid linux route file"),
		},
	}
	for _, test := range tests {
		result, err := parseToLinuxRouteStruct(test.input)
		assert.Equal(t, test.err, err)
		assert.Equal(t, test.expected, result)
	}
}

func TestParseNetstatToRouteStruct(t *testing.T) {
	tests := []struct {
		input    []byte
		expected routeStruct
		err      error
	}{
		{
			// // $ netstat -nr
			// want en0
			// From man netstat:
			// U       RTF_UP           Route usable
			// G       RTF_GATEWAY      Destination requires forwarding by intermediary
			// S       RTF_STATIC       Manually added
			// c       RTF_PRCLONING    Protocol-specified generate new routes on use
			// I       RTF_IFSCOPE      Route is associated with an interface scope
			// so if multiple Gatwways and route is not associated with an interface scope
			input: []byte(`
				Routing tables
				Internet:
				Destination        Gateway            Flags        Netif Expire
				default            10.0.0.1           UGSc           en0        
				default            10.0.0.1           UGScI          en1`),
			expected: routeStruct{
				Iface:       "en0",
				Destination: "default",
				Gateway:     "10.0.0.1",
				Flags:       "UGSc",
			},
			err: nil,
		},
		{
			// still want en0
			input: []byte(`
				Routing tables
				Internet:
				Destination        Gateway            Flags        Netif Expire
				default            10.0.0.1           UGScI          en1
				default            10.0.0.1           UGSc           en0`),
			expected: routeStruct{
				Iface:       "en0",
				Destination: "default",
				Gateway:     "10.0.0.1",
				Flags:       "UGSc",
			},
			err: nil,
		},
		{
			input: []byte(`
				Routing tables
			
				Internet:
				Destination        Gateway            Flags      Netif Expire
				default            10.88.88.2         UGS         em0
				10.88.88.0/24      link#1             U           em0
				10.88.88.148       link#1             UHS         lo0
				127.0.0.1          link#2             UH          lo0`),
			expected: routeStruct{
				Iface:       "em0",
				Destination: "default",
				Gateway:     "10.88.88.2",
				Flags:       "UGS",
			},
			err: nil,
		},
		{
			input: []byte(`
				Routing tables
				Internet:
				Destination        Gateway            Flags        Netif Expire`),
			expected: routeStruct{},
			err:      errors.New("unable to determine default interface"),
		},
		{
			input:    []byte(``),
			expected: routeStruct{},
			err:      errors.New("unable to determine default interface"),
		},
	}
	for _, test := range tests {
		result, err := parseNetstatToRouteStruct(test.input)
		assert.Equal(t, test.err, err)
		assert.Equal(t, test.expected, result)
	}
}

func TestParseSbinRoute(t *testing.T) {
	tests := []struct {
		input    []byte
		expected routeStruct
		err      error
	}{
		{
			input: []byte(`
				route to: default
				destination: default
					mask: default
					gateway: 192.168.0.1
				interface: en0
					flags: <UP,GATEWAY,DONE,STATIC,PRCLONING,GLOBAL>
				recvpipe  sendpipe  ssthresh  rtt,msec    rttvar  hopcount      mtu     expire
					0         0         0         0         0         0      1500         0 `),
			expected: routeStruct{
				Iface:       "en0",
				Destination: "default",
				Gateway:     "192.168.0.1",
				Flags:       "<UP,GATEWAY,DONE,STATIC,PRCLONING,GLOBAL>",
				Mask:        "default",
			},
			err: nil,
		},
		{
			input:    []byte(`not standard`),
			expected: routeStruct{},
			err:      errors.New("unable to determine default interface"),
		},
		{
			input:    []byte(``),
			expected: routeStruct{},
			err:      errors.New("unable to determine default interface"),
		},
	}
	for _, test := range tests {
		result, err := parseToSbinRouteStruct(test.input)
		assert.Equal(t, test.err, err)
		assert.Equal(t, test.expected, result)
	}
}
