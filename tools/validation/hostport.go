// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package validation provides functions to verify the correctness of common
// input values such as ports and hosts.
package validation

import (
	"errors"
	"net"
	"net/netip"
	"strconv"

	"golang.org/x/net/http/httpguts"
	"golang.org/x/net/idna"
)

// ValidateHost checks whether the given string is a valid host.
// It accepts IPv4, IPv6, ASCII hostnames, and IDNs, and rejects hosts
// containing ports or invalid characters.
func ValidateHost(host string) error {
	if addr, err := netip.ParseAddr(host); err == nil {
		if addr.Zone() != "" {
			return errors.New("host cannot contain a zone")
		}
		return nil
	}
	if _, port, err := net.SplitHostPort(host); err == nil {
		if _, err = strconv.ParseUint(port, 10, 64); err == nil {
			return errors.New("host cannot include a port")
		}
		return errors.New("host is not valid")
	}
	if !httpguts.ValidHostHeader(host) {
		var err error
		host, err = idna.Lookup.ToASCII(host)
		if err != nil {
			return errors.New("host is not valid")
		}
	}
	if n := len(host); n == 0 || n > 253 {
		return errors.New("host length in bytes must be in range [1,253]")
	}
	return nil
}

// ValidatePort checks whether the given integer is a valid TCP port.
// Valid ports are in the range [1, 65535].
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return errors.New("port must be in range [1,65535]")
	}
	return nil
}

// ValidatePortString checks whether the given string is a valid TCP port.
// It returns the port as an integer and nil if valid.
// If invalid, it returns 0 and an error.
// Valid ports are in the range [1, 65535].
func ValidatePortString(port string) (int, error) {
	if port == "" {
		return 0, errors.New("port cannot be empty")
	}
	p := 0
	for i := 0; i < len(port); i++ {
		c := port[i]
		if c < '0' || c > '9' {
			return 0, errors.New("port is not a positive integer")
		}
		p = p*10 + int(c-'0')
		if p > 65535 {
			return 0, errors.New("port must not exceed 65535")
		}
	}
	if p == 0 {
		return 0, errors.New("port cannot be 0")
	}
	return p, nil
}
