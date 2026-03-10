/*
Copyright 2026 Swisscom (Schweiz) AG.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"testing"
	"time"
)

func TestIpsInRange_SingleIPv4(t *testing.T) {
	ips, err := ipsInRange("10.0.0.1", "10.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 1 || ips[0] != "10.0.0.1" {
		t.Errorf("expected [10.0.0.1], got %v", ips)
	}
}

func TestIpsInRange_MultipleIPv4(t *testing.T) {
	ips, err := ipsInRange("192.168.0.1", "192.168.0.5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"192.168.0.1", "192.168.0.2", "192.168.0.3", "192.168.0.4", "192.168.0.5"}
	if len(ips) != len(expected) {
		t.Fatalf("expected %d IPs, got %d", len(expected), len(ips))
	}
	for i, ip := range ips {
		if ip != expected[i] {
			t.Errorf("index %d: expected %s, got %s", i, expected[i], ip)
		}
	}
}

func TestIpsInRange_IPv4CrossingOctetBoundary(t *testing.T) {
	ips, err := ipsInRange("10.0.0.254", "10.0.1.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"10.0.0.254", "10.0.0.255", "10.0.1.0", "10.0.1.1", "10.0.1.2"}
	if len(ips) != len(expected) {
		t.Fatalf("expected %d IPs, got %d", len(expected), len(ips))
	}
	for i, ip := range ips {
		if ip != expected[i] {
			t.Errorf("index %d: expected %s, got %s", i, expected[i], ip)
		}
	}
}

func TestIpsInRange_SingleIPv6(t *testing.T) {
	ips, err := ipsInRange("fd00::1", "fd00::1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 1 || ips[0] != "fd00::1" {
		t.Errorf("expected [fd00::1], got %v", ips)
	}
}

func TestIpsInRange_MultipleIPv6(t *testing.T) {
	ips, err := ipsInRange("fd00::1", "fd00::5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"fd00::1", "fd00::2", "fd00::3", "fd00::4", "fd00::5"}
	if len(ips) != len(expected) {
		t.Fatalf("expected %d IPs, got %d", len(expected), len(ips))
	}
	for i, ip := range ips {
		if ip != expected[i] {
			t.Errorf("index %d: expected %s, got %s", i, expected[i], ip)
		}
	}
}

func TestIpsInRange_IPv6CrossingSegmentBoundary(t *testing.T) {
	ips, err := ipsInRange("fd00::fffe", "fd00::1:1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"fd00::fffe", "fd00::ffff", "fd00::1:0", "fd00::1:1"}
	if len(ips) != len(expected) {
		t.Fatalf("expected %d IPs, got %d", len(expected), len(ips))
	}
	for i, ip := range ips {
		if ip != expected[i] {
			t.Errorf("index %d: expected %s, got %s", i, expected[i], ip)
		}
	}
}

func TestIpsInRange_ErrorMixedIPv4StartIPv6End(t *testing.T) {
	_, err := ipsInRange("10.0.0.1", "fd00::1")
	if err == nil {
		t.Fatal("expected error for mixed IP versions, got nil")
	}
}

func TestIpsInRange_ErrorMixedIPv6StartIPv4End(t *testing.T) {
	_, err := ipsInRange("fd00::1", "10.0.0.1")
	if err == nil {
		t.Fatal("expected error for mixed IP versions, got nil")
	}
}

func TestIpsInRange_ErrorInvalidStartAddress(t *testing.T) {
	_, err := ipsInRange("not-an-ip", "10.0.0.1")
	if err == nil {
		t.Fatal("expected error for invalid start address, got nil")
	}
}

func TestIpsInRange_MixedIPFamilies(t *testing.T) {
	_, err := ipsInRange("192.168.1.1", "::ffff:c0a8:102")
	if err == nil {
		t.Fatal("expected error for mixed ip families, got nil")
	}
}

func TestIpsInRange_ErrorInvalidEndAddress(t *testing.T) {
	_, err := ipsInRange("10.0.0.1", "not-an-ip")
	if err == nil {
		t.Fatal("expected error for invalid end address, got nil")
	}
}

func TestIpsInRange_ErrorStartGreaterThanEnd(t *testing.T) {
	_, err := ipsInRange("10.0.0.5", "10.0.0.1")
	if err == nil {
		t.Fatal("expected error when start address is greater than end address, got nil")
	}
}

func TestIpsInRange_ErrorStartGreaterThanEndIPv6(t *testing.T) {
	_, err := ipsInRange("fd00::5", "fd00::1")
	if err == nil {
		t.Fatal("expected error when start address is greater than end address, got nil")
	}
}

func TestIpsInRange_IPv4MappedMixedWithPureIPv4(t *testing.T) {
	// One address in IPv4-mapped IPv6 notation, the other in plain IPv4.
	// These should be treated as different IP versions because
	// the string formats differ.
	_, err := ipsInRange("::ffff:10.0.0.1", "10.0.0.5")
	if err == nil {
		t.Fatal("expected error mixing IPv4-mapped IPv6 with plain IPv4, got nil")
	}
}

func TestIpsInRange_ErrorBothInvalid(t *testing.T) {
	_, err := ipsInRange("abc", "xyz")
	if err == nil {
		t.Fatal("expected error for both invalid addresses, got nil")
	}
}

func TestIpsInRange_EmptyStrings(t *testing.T) {
	_, err := ipsInRange("", "")
	if err == nil {
		t.Fatal("expected error for empty strings, got nil")
	}
}

func TestIpsInRange_IPv4MaxAddressOverflow(t *testing.T) {
	// When the end address is the maximum IPv4 address, incrementIP overflows
	// to 0.0.0.0, causing an infinite loop. This test guards against that.
	done := make(chan struct{})
	go func() {
		defer close(done)
		ips, err := ipsInRange("255.255.255.254", "255.255.255.255")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		expected := []string{"255.255.255.254", "255.255.255.255"}
		if len(ips) != len(expected) {
			t.Errorf("expected %d IPs, got %d", len(expected), len(ips))
			return
		}
		for i, ip := range ips {
			if ip != expected[i] {
				t.Errorf("index %d: expected %s, got %s", i, expected[i], ip)
			}
		}
	}()
	select {
	case <-done:
		// success
	case <-time.After(3 * time.Second):
		t.Fatal("ipsInRange did not return within 3 seconds — likely infinite loop due to overflow at max IPv4 address")
	}
}
func TestIpsInRange_IPv6MaxAddressOverflow(t *testing.T) {
	// When the end address is the maximum IPv4 address, incrementIP overflows
	// to 0.0.0.0, causing an infinite loop. This test guards against that.
	done := make(chan struct{})
	go func() {
		defer close(done)
		ips, err := ipsInRange("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		expected := []string{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"}
		if len(ips) != len(expected) {
			t.Errorf("expected %d IPs, got %d", len(expected), len(ips))
			return
		}
		for i, ip := range ips {
			if ip != expected[i] {
				t.Errorf("index %d: expected %s, got %s", i, expected[i], ip)
			}
		}
	}()
	select {
	case <-done:
		// success
	case <-time.After(3 * time.Second):
		t.Fatal("ipsInRange did not return within 3 seconds — likely infinite loop due to overflow at max IPv4 address")
	}
}
