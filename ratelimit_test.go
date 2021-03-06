package ratelimit

import (
	"sync"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
)

func Test_rateLimitOptions_refreshWindows(t *testing.T) {
	t.Run("Should refresh", func(t *testing.T) {
		RateLimit := RateLimit{
			WindowLength: 15 * 60,
			currentWindow: &RequestCountTracker{
				requestCount: map[string]int64{},
				startTime:    time.Now().Add(-35 * time.Minute),
				endTime:      time.Now().Add(-20 * time.Minute),
				_mutex:       &sync.RWMutex{},
			},
		}

		if didRefresh := RateLimit.refreshWindows(); !didRefresh {
			t.Errorf("Should have refreshed, but did not: %+v", RateLimit)
		}
	})

	t.Run("Should not refresh", func(t *testing.T) {
		RateLimit := RateLimit{
			WindowLength: 60,
			currentWindow: &RequestCountTracker{
				requestCount: map[string]int64{},
				startTime:    time.Now().Add(-30 * time.Minute),
				endTime:      time.Now().Add(30 * time.Minute),
				_mutex:       &sync.RWMutex{},
			},
		}

		if didRefresh := RateLimit.refreshWindows(); didRefresh {
			t.Errorf("Should not have refreshed, but did: %+v", RateLimit)
		}
	})
}

func Test_rateLimitOptions_blockingAndRequestCounting(t *testing.T) {

	// in this we test the case described in the documentation for
	// getInterpolatedRequestCount()
	hostName := "10.0.0.127"

	rl := RateLimit{
		ByHeader:     "",
		WindowLength: caddy.Duration(20 * time.Minute),
		MaxRequests:  200,
	}
	rl.Provision(caddy.Context{})

	rl.currentWindow.requestCount[hostName] = 100
	rl.currentWindow.startTime = rl.currentWindow.startTime.Add(-10 * time.Minute)
	rl.currentWindow.endTime = rl.currentWindow.endTime.Add(-10 * time.Minute)

	// start/end time doesn't really matter for previous window
	rl.previousWindow = &RequestCountTracker{
		requestCount: map[string]int64{hostName: 50},
		_mutex:       &sync.RWMutex{},
	}

	t.Run("50-50 split should interpolate to 75 requests", func(t *testing.T) {
		// expected result is (100+50) / 2
		if requestCount := rl.getInterpolatedRequestCount(hostName); requestCount != 75 {
			t.Errorf("Expected requestCount of 75 for 50-50 split, got %v", requestCount)
		}
	})

	t.Run("50-50 split should not block as 76 < 100", func(t *testing.T) {
		if shouldBlock := rl.requestShouldBlock(hostName); shouldBlock {
			t.Errorf("Well clear of max request count, should not block, got %+v", rl)
		}

	})

	// test whether blocking works
	rl.MaxRequests = 50
	t.Run("50-50 split should block with now reduced maxRequest as 77 > 50", func(t *testing.T) {
		if shouldBlock := rl.requestShouldBlock(hostName); !shouldBlock {
			t.Errorf("Should have blocked with reduced maxRequests, did not, got %+v", rl)
		}
	})
}

func Test_rateLimitOptions_blockingAndRequestCountingByHeader(t *testing.T) {

	header := "Authorization"

	rl := RateLimit{
		ByHeader:     "",
		WindowLength: caddy.Duration(20 * time.Minute),
		MaxRequests:  200,
	}
	rl.Provision(caddy.Context{})

	rl.currentWindow.requestCount[header] = 100
	rl.currentWindow.startTime = rl.currentWindow.startTime.Add(-10 * time.Minute)
	rl.currentWindow.endTime = rl.currentWindow.endTime.Add(-10 * time.Minute)

	// start/end time doesn't really matter for previous window
	rl.previousWindow = &RequestCountTracker{
		requestCount: map[string]int64{header: 50},
		_mutex:       &sync.RWMutex{},
	}

	t.Run("50-50 split should interpolate to 75 requests", func(t *testing.T) {
		// expected result is (100+50) / 2
		if requestCount := rl.getInterpolatedRequestCount(header); requestCount != 75 {
			t.Errorf("Expected requestCount of 75 for 50-50 split, got %v", requestCount)
		}
	})

	t.Run("50-50 split should not block as 76 < 100", func(t *testing.T) {
		if shouldBlock := rl.requestShouldBlock(header); shouldBlock {
			t.Errorf("Well clear of max request count, should not block, got %+v", rl)
		}

	})

	// test whether blocking works
	rl.MaxRequests = 50
	t.Run("50-50 split should block with now reduced maxRequest as 77 > 50", func(t *testing.T) {
		if shouldBlock := rl.requestShouldBlock(header); !shouldBlock {
			t.Errorf("Should have blocked with reduced maxRequests, did not, got %+v", rl)
		}
	})
}

func Test_rateLimitOptions_setupRateLimit(t *testing.T) {
	t.Run("Should initialise properly", func(t *testing.T) {
		rl := RateLimit{
			ByHeader:     "",
			WindowLength: caddy.Duration(60 * time.Minute),
			MaxRequests:  200,
		}
		rl.Provision(caddy.Context{})

		hostName := "localhost"
		if count := rl.currentWindow.getRequestCountFor(hostName); count != 0 {
			t.Errorf("Unexpected request count - expected 0, got %v", count)
		}

		if count := rl.previousWindow.getRequestCountFor(hostName); count != 0 {
			t.Errorf("Unexpected request count - expected 0, got %v", count)
		}
	})

	// value of window kept very low for testing, minimum value should be 5 minutes
	t.Run("Should shuffle windows after one second", func(t *testing.T) {
		rl := RateLimit{
			ByHeader:     "",
			WindowLength: caddy.Duration(1 * time.Second),
			MaxRequests:  1000,
		}
		rl.Provision(caddy.Context{})

		hostName := "localhost"

		rl.requestShouldBlock(hostName)

		if count := rl.currentWindow.getRequestCountFor(hostName); count != 1 {
			t.Errorf("Unexpected request count - expected 1, got %v", count)
		}

		time.Sleep(1100 * time.Millisecond)

		if count := rl.currentWindow.getRequestCountFor(hostName); count != 0 {
			t.Errorf("Unexpected request count - expected 0, got %v", count)
		}

		if count := rl.previousWindow.getRequestCountFor(hostName); count != 1 {
			t.Errorf("Unexpected request count - expected 1, got %v", count)
		}
	})
}
