/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package network

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2/proto"
)

const (
	IPv6len   = 16
	hexDigit  = "0123456789abcdef"
	linuxFile = "/proc/net/route"
	FREEBSD   = "freebsd"
	SOLARIS   = "solaris"
	DARWIN    = "darwin"
	LINUX     = "linux"
)

var (
	reOverflow                 = regexp.MustCompile(`\s*(\d+)\s*`)
	reTimesOverflowed          = regexp.MustCompile("times the listen queue of a socket overflowed")
	reTimesOverflowedWithNstat = regexp.MustCompile("TcpExtListenOverflows")
)

type routeStruct struct {
	Iface       string
	Destination string
	Gateway     string
	Flags       string
	RefCnt      string
	Use         string
	Metric      string
	Mask        string
	MTU         string
	Window      string
	IRTT        string
}

// Get net overflow. The command (netstat) to get net overflow may not be available on all platforms
func GetNetOverflow() (float64, error) {
	overflows := 0.0
	switch runtime.GOOS {
	case FREEBSD, SOLARIS:
		return getNetOverflowCmd("netstat", "-s", reTimesOverflowed, overflows)
	case DARWIN:
		return overflows, errors.New("this operating system is not implemented")
	case LINUX:
		return getNetOverflowCmd("nstat", "-az", reTimesOverflowedWithNstat, overflows)
	default:
		return overflows, errors.New("this operating system is not implemented")
	}
}

func getNetOverflowCmd(cmd string, flags string, pattern *regexp.Regexp, overflows float64) (float64, error) {
	netstatCmd := exec.Command(cmd, flags)
	outbuf, err := netstatCmd.CombinedOutput()

	if err != nil {
		errMsg := fmt.Sprintf("%s not available: %v", cmd, err)
		log.Debug(errMsg)
		return overflows, errors.New(errMsg)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(outbuf)))
	matches := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if pattern.MatchString(line) {
			matches = append(matches, line)

		}
	}
	so := strings.Join(matches, "\n")

	ofm := reOverflow.FindStringSubmatch(so)
	if len(ofm) > 1 {
		overflows, _ = strconv.ParseFloat(ofm[1], 64)
	}

	return overflows, nil
}

func GetDataplaneNetworks() (res *proto.Network) {
	const (
		NetmaskFormat = "%v.%v.%v.%v"
	)
	ifs, err := net.Interfaces()
	if err != nil {
		log.Errorf("error getting network interfaces on host: %v", err)
		return &proto.Network{}
	}

	interfaces := []*proto.NetworkInterface{}
	for _, netInterface := range ifs {
		networkInterface := &proto.NetworkInterface{
			Mac:  netInterface.HardwareAddr.String(),
			Name: netInterface.Name,
		}
		ipv4Addrs := make([]*proto.Address, 0)
		ipv6Addrs := make([]*proto.Address, 0)

		addrs, err := netInterface.Addrs()
		if err != nil || len(addrs) == 0 {
			// don't care about things without addrs
			continue
		}
		for _, a := range addrs {
			v, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			mask, _ := net.IPMask.Size(v.Mask)

			addr := &proto.Address{}
			addr.Address = v.IP.String()
			addr.Prefixlen = int64(mask)

			if v.IP.To4() != nil {
				addr.Netmask = fmt.Sprintf(NetmaskFormat, v.Mask[0], v.Mask[1], v.Mask[2], v.Mask[3])
				ipv4Addrs = append(ipv4Addrs, addr)
			} else {
				addr.Netmask = ipv6ToStr(v.Mask)
				ipv6Addrs = append(ipv6Addrs, addr)
			}
		}
		networkInterface.Ipv4 = ipv4Addrs
		networkInterface.Ipv6 = ipv6Addrs
		interfaces = append(interfaces, networkInterface)
	}

	defaultNetworkInterface, err := getDefaultNetworkInterfaceCrossPlatform()
	if err != nil {
		log.Debugf("Error getting default network interface, %v", err)
	}

	if defaultNetworkInterface == "" && len(ifs) > 0 {
		defaultNetworkInterface = ifs[0].Name
	}
	return &proto.Network{Interfaces: interfaces, Default: defaultNetworkInterface}
}

func getDefaultNetworkInterfaceCrossPlatform() (string, error) {
	const (
		SBinRoute        = "/sbin/route"
		SBinFlags        = "-n"
		SBinCommand      = "get"
		SBinDefaultRoute = "0.0.0.0"
		Netstat          = "netstat"
		NetstatFlags     = "-rn"
	)
	switch runtime.GOOS {
	case FREEBSD:
		return getInterfaceUsing(Netstat, NetstatFlags)
	case SOLARIS:
		return getInterfaceUsing(Netstat, NetstatFlags)
	case DARWIN:
		routeCmd := exec.Command(SBinRoute, SBinFlags, SBinCommand, SBinDefaultRoute)
		output, err := routeCmd.CombinedOutput()
		if err != nil {
			return "", err
		}
		routeStruct, err := parseToSbinRouteStruct(output)
		if err != nil {
			return "", err
		}
		return routeStruct.Iface, nil
	case LINUX:
		f, err := os.Open(linuxFile)
		if err != nil {
			return "", fmt.Errorf("Can't access %s", linuxFile)
		}
		defer f.Close()

		output, err := ioutil.ReadAll(f)
		if err != nil {
			return "", fmt.Errorf("Can't read contents of %s", linuxFile)
		}

		parsedStruct, err := parseToLinuxRouteStruct(output)
		if err != nil {
			return "", err
		}

		return parsedStruct.Iface, nil
	default:
		return "", errors.New("this operating system is not implemented")
	}
}

func getInterfaceUsing(netstat string, flags string) (string, error) {
	netstatCmd := exec.Command(netstat, flags)
	output, err := netstatCmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	routeStruct, err := parseNetstatToRouteStruct(output)
	if err != nil {
		return "", err
	}
	return routeStruct.Iface, nil
}

func parseToSbinRouteStruct(output []byte) (routeStruct, error) {
	const (
		DestinationStr = "destination:"
		MaskStr        = "mask:"
		GatewayStr     = "gateway:"
		InterfaceStr   = "interface:"
		FlagsStr       = "flags:"
	)
	var err error
	rs := routeStruct{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			switch index := fields[0]; index {
			case DestinationStr:
				rs.Destination = fields[1]
			case MaskStr:
				rs.Mask = fields[1]
			case GatewayStr:
				rs.Gateway = fields[1]
			case InterfaceStr:
				rs.Iface = fields[1]
			case FlagsStr:
				rs.Flags = fields[1]
			default:
				continue
			}
		}
	}
	if rs.Iface == "" {
		err = errors.New("unable to determine default interface")
	} else {
		err = nil
	}

	return rs, err
}

func parseNetstatToRouteStruct(output []byte) (routeStruct, error) {
	const (
		destinationField = 0
		gatewayField     = 1
		flagsField       = 2
		interfaceField   = 3
		interfaceFlag    = "I"
		searchString     = "default"
	)
	outputLines := strings.Split(string(output), "\n")
	for _, line := range outputLines {
		fields := strings.Fields(line)
		// this check prevents hitting the first 3 lines of nestat output
		if len(fields) >= 2 && fields[destinationField] == searchString {
			if !strings.ContainsAny(fields[flagsField], interfaceFlag) {
				return routeStruct{
					Iface:       fields[interfaceField],
					Destination: fields[destinationField],
					Gateway:     fields[gatewayField],
					Flags:       fields[flagsField],
				}, nil
			}
		}
	}

	return routeStruct{}, errors.New("unable to determine default interface")
}

// Referenced from https://github.com/jackpal/gateway and adapted
func parseToLinuxRouteStruct(output []byte) (routeStruct, error) {
	const (
		destinationField = 1 // field containing hex destination address
	)
	scanner := bufio.NewScanner(bytes.NewReader(output))

	// Skip header line
	if !scanner.Scan() {
		return routeStruct{}, errors.New("Invalid linux route file")
	}

	for scanner.Scan() {
		row := scanner.Text()
		tokens := strings.Fields(strings.TrimSpace(row))
		if len(tokens) < 11 {
			return routeStruct{}, fmt.Errorf("invalid row '%s' in route file: doesn't have 11 fields", row)
		}

		// Cast hex destination address to int
		destinationHex := "0x" + tokens[destinationField]
		destination, err := strconv.ParseInt(destinationHex, 0, 64)
		if err != nil {
			return routeStruct{}, fmt.Errorf(
				"parsing destination field hex '%s' in row '%s': %w",
				destinationHex,
				row,
				err,
			)
		}

		// The default interface is the one that's 0
		if destination != 0 {
			continue
		}

		return routeStruct{
			Iface:       tokens[0],
			Destination: tokens[1],
			Gateway:     tokens[2],
			Flags:       tokens[3],
			RefCnt:      tokens[4],
			Use:         tokens[5],
			Metric:      tokens[6],
			Mask:        tokens[7],
			MTU:         tokens[8],
			Window:      tokens[9],
			IRTT:        tokens[10],
		}, nil
	}
	return routeStruct{}, errors.New("interface with default destination not found")
}

func ipv6ToStr(ip []byte) string {
	p := ip
	// Find longest run of zeros.
	e0 := -1
	e1 := -1
	for i := 0; i < IPv6len; i += 2 {
		j := i
		for j < IPv6len && p[j] == 0 && p[j+1] == 0 {
			j += 2
		}
		if j > i && j-i > e1-e0 {
			e0 = i
			e1 = j
			i = j
		}
	}
	// The symbol "::" MUST NOT be used to shorten just one 16 bit 0 field.
	if e1-e0 <= 2 {
		e0 = -1
		e1 = -1
	}

	const maxLen = len("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")
	b := make([]byte, 0, maxLen)

	// Return with possible :: in place of run of zeros
	for i := 0; i < IPv6len; i += 2 {
		if i == e0 {
			b = append(b, ':', ':')
			i = e1
			if i >= IPv6len {
				break
			}
		} else if i > 0 {
			b = append(b, ':')
		}
		b = appendHex(b, (uint32(p[i])<<8)|uint32(p[i+1]))
	}
	return string(b)
}

func appendHex(dst []byte, i uint32) []byte {
	if i == 0 {
		return append(dst, '0')
	}
	for j := 7; j >= 0; j-- {
		v := i >> uint(j*4)
		if v > 0 {
			dst = append(dst, hexDigit[v&0xf])
		}
	}
	return dst
}
