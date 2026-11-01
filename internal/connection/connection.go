// Package connection probes emulator ADB endpoints.
//
// This is a reachability check, not a full MaaCore connection handshake. It is
// enough to validate the address configured in the UI and to suggest common
// emulator ports during first-time setup.
package connection

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

// Candidate is a reachable ADB endpoint.
type Candidate struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	LatencyMs int    `json:"latencyMs"`
}

// TestResult is the outcome of testing one address.
type TestResult struct {
	OK        bool   `json:"ok"`
	Address   string `json:"address"`
	LatencyMs int    `json:"latencyMs"`
	Message   string `json:"message"`
}

type endpoint struct {
	Name    string
	Address string
}

var knownEndpoints = []endpoint{
	{"MuMu 模拟器", "127.0.0.1:16384"},
	{"MuMu 模拟器（实例 1）", "127.0.0.1:16385"},
	{"MuMu 模拟器（实例 2）", "127.0.0.1:16416"},
	{"MuMu 模拟器（旧版）", "127.0.0.1:7555"},
	{"雷电模拟器", "127.0.0.1:5555"},
	{"雷电模拟器（实例 1）", "127.0.0.1:5556"},
	{"雷电模拟器（实例 2）", "127.0.0.1:5557"},
	{"蓝叠模拟器", "127.0.0.1:5565"},
	{"夜神模拟器", "127.0.0.1:62001"},
	{"夜神模拟器（实例 1）", "127.0.0.1:62025"},
	{"夜神模拟器（实例 2）", "127.0.0.1:62026"},
	{"逍遥模拟器", "127.0.0.1:21503"},
	{"Windows Subsystem for Android", "127.0.0.1:58526"},
}

// Detect scans common emulator ADB ports.
func Detect(ctx context.Context) []Candidate {
	type result struct {
		candidate Candidate
	}

	results := make(chan result, len(knownEndpoints))
	var wg sync.WaitGroup
	for _, item := range knownEndpoints {
		wg.Add(1)
		go func(item endpoint) {
			defer wg.Done()
			start := time.Now()
			dialer := net.Dialer{Timeout: 250 * time.Millisecond}
			conn, err := dialer.DialContext(ctx, "tcp", item.Address)
			if err != nil {
				return
			}
			_ = conn.Close()
			results <- result{candidate: Candidate{
				Name:      item.Name,
				Address:   item.Address,
				LatencyMs: int(time.Since(start).Milliseconds()),
			}}
		}(item)
	}

	wg.Wait()
	close(results)

	seen := map[string]struct{}{}
	candidates := make([]Candidate, 0, len(knownEndpoints))
	for item := range results {
		if _, exists := seen[item.candidate.Address]; exists {
			continue
		}
		seen[item.candidate.Address] = struct{}{}
		candidates = append(candidates, item.candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].LatencyMs < candidates[j].LatencyMs
	})
	return candidates
}

// Test checks one address. A bare host gets the standard 5555 port.
func Test(ctx context.Context, address string) TestResult {
	address = normalizeAddress(address)
	if address == "" {
		return TestResult{OK: false, Message: "请先填写连接地址"}
	}

	start := time.Now()
	dialer := net.Dialer{Timeout: 1200 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return TestResult{
			OK:      false,
			Address: address,
			Message: fmt.Sprintf("连接失败：%v", err),
		}
	}
	_ = conn.Close()

	latency := int(time.Since(start).Milliseconds())
	return TestResult{
		OK:        true,
		Address:   address,
		LatencyMs: latency,
		Message:   fmt.Sprintf("连接成功，延迟 %d ms", latency),
	}
}

func normalizeAddress(address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return ""
	}
	if !strings.Contains(address, ":") {
		return net.JoinHostPort(address, "5555")
	}
	return address
}
