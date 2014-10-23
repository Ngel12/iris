// Iris - Decentralized cloud messaging
// Copyright (c) 2014 Project Iris. All rights reserved.
//
// Iris is dual licensed: you can redistribute it and/or modify it under the
// terms of the GNU General Public License as published by the Free Software
// Foundation, either version 3 of the License, or (at your option) any later
// version.
//
// The framework is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
// more details.
//
// Alternatively, the Iris framework may be used in accordance with the terms
// and conditions contained in a signed written agreement between you and the
// author(s).

package bootstrap

import (
	"net"
	"testing"
	"time"

	"gopkg.in/inconshreveable/log15.v2"
)

// Tests that the probing ad-hoc seeder indeed generates IP addresses in the
// correct range for well formed subnets.
func TestProbeSeeder(t *testing.T) {
	addr, _ := net.ResolveIPAddr("ip", "192.168.0.100")
	for subnet := 30; subnet >= 22; subnet-- {
		testProbeSeeder(t, subnet, addr)
	}
}

// Tests that the probing ad-hoc seeder indeed generates IP addresses in the
// correct range for a specific ipnet configuration.
func testProbeSeeder(t *testing.T, subnet int, addr *net.IPAddr) {
	// Create the IP net from the configurations
	ipnet := &net.IPNet{
		IP:   addr.IP,
		Mask: net.CIDRMask(subnet, 32),
	}
	// Create the probing seed generator
	seeder, err := newProbeSeeder(ipnet, log15.New("ipnet", ipnet))
	if err != nil {
		t.Fatalf("failed to create seed generator: %v.", err)
	}
	// Create a seed sink and start generating peers
	sink := make(chan *net.IPAddr)
	phase := uint32(0)

	if err := seeder.Start(sink, &phase); err != nil {
		t.Fatalf("failed to start seed generator: %v.", err)
	}
	// Retrieve a large batch of random addresses, ensuring they are in range
	iters := 100000
	addrs := make(map[string]int)
	for i := 0; i < iters; i++ {
		select {
		case addr := <-sink:
			if !ipnet.Contains(addr.IP) {
				t.Fatalf("out of range address generated: %v.", addr)
			}
			addrs[addr.String()]++
		case <-time.After(time.Second):
			t.Fatalf("failed to retrieve next address")
		}
	}
	// Verify that multipliers are within expected range
	for _, count := range addrs {
		lo := (iters / ((1 << uint(32-subnet)) - 2)) / 10 * 5
		hi := (iters / ((1 << uint(32-subnet)) - 2)) / 10 * 15
		if lo > count || count > hi {
			t.Fatalf("non uniform address count: have %v, want in [%v-%v].", count, lo, hi)
		}
	}
	// Terminate the generator
	if err := seeder.Close(); err != nil {
		t.Fatalf("failed to terminate seed generator: %v.", err)
	}
}

// Tests two particular cases of network configurations where the host space is
// empty (used during point-to-point connections).
func TestProbeSeederEmpyHostSpace(t *testing.T) {
	addr, _ := net.ResolveIPAddr("ip", "192.168.0.100")
	for subnet := 32; subnet >= 31; subnet-- {
		testProbeSeederEmpyHostSpace(t, subnet, addr)
	}
}

// Tests that the scanning ad-hoc seeder indeed generates IP addresses in the
// correct order and range for a specific ipnet configuration.
func testProbeSeederEmpyHostSpace(t *testing.T, subnet int, addr *net.IPAddr) {
	// Create the IP net from the configurations
	ipnet := &net.IPNet{
		IP:   addr.IP,
		Mask: net.CIDRMask(subnet, 32),
	}
	// Create the probing seed generator
	seeder, err := newProbeSeeder(ipnet, log15.New("ipnet", ipnet))
	if err != nil {
		t.Fatalf("failed to create seed generator: %v.", err)
	}
	// Create a seed sink and start generating peers
	sink := make(chan *net.IPAddr)
	phase := uint32(0)

	if err := seeder.Start(sink, &phase); err != nil {
		t.Fatalf("failed to start seed generator: %v.", err)
	}
	// Make sure no hosts are generated
	select {
	case addr := <-sink:
		t.Fatalf("unexpected host generated: %v.", addr)
	case <-time.After(10 * time.Millisecond):
	}
	// Terminate the generator
	seeder.Close()
}
