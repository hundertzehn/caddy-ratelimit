package ratelimit

import (
	"fmt"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	"github.com/caddyserver/caddy/v2"
)

func init() {
	caddy.RegisterModule(RateLimit{})
	// register a plugin that can load the Caddyfile when Caddy starts
	httpcaddyfile.RegisterHandlerDirective("rate_limit", parseRateLimit)
}

// rateLimitOptions stores options detailing how rate limiting should be applied,
// as well as the current and previous window's key:requestCount mapping
type RateLimit struct {
	ByHeader string `json:"by_header,omitempty"`

	// window length for request rate checking (>= 1 minute)
	WindowLength int64 `json:"window_length"`

	// max request that should be processed per key in a given windowDuration
	MaxRequests int64 `json:"max_requests"`

	// current window's request count per key
	currentWindow *RequestCountTracker

	// previous window's request count per key
	previousWindow *RequestCountTracker
}

func (rl *RateLimit) Provision(_ctx caddy.Context) error {
	if nil == rl.currentWindow {
		rl.currentWindow = newRequestCountTracker(rl.windowDuration())
		rl.previousWindow = &RequestCountTracker{}

		go func() { // automatic shuffling of request count tracking windows
			for {
				time.Sleep(rl.currentWindow.endTime.Sub(time.Now()))
				rl.refreshWindows()
			}
		}()
	}
	return nil
}

func (rl *RateLimit) windowDuration() time.Duration {
	return time.Duration(rl.WindowLength) * time.Second
}

// CaddyModule returns the Caddy module information.
func (RateLimit) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.ratelimit",
		New: func() caddy.Module { return new(RateLimit) },
	}
}

func (rl *RateLimit) isByHost() bool {
	return 0 == len(rl.ByHeader)
}

// refreshWindows() checks if currentWindow has reached its expiry time, and if it has,
// moves currentWindow to previousWindow, and re-initialises currentWindow
func (rl *RateLimit) refreshWindows() (didRefresh bool) {
	if rl.currentWindow.endTime.Before(time.Now()) {
		rl.previousWindow = rl.currentWindow
		rl.currentWindow = newRequestCountTracker(rl.windowDuration())

		didRefresh = true
	}
	return
}

// requestShouldBlock checks whether the request from a given key name should block,
// and increments the request counter for the key first
// will block if current request would push the key over the blocking threshold
func (rl *RateLimit) requestShouldBlock(key string) (shouldBlock bool) {
	rl.currentWindow.addRequestFor(key)                         // increment request counter for the key
	return rl.getInterpolatedRequestCount(key) > rl.MaxRequests // check if they now are above the request limit
}

// getInterpolatedRequestCount gets an interpolated request count for a specified key
// Always considers requests across the given windowDuration
// More details: https://blog.cloudflare.com/counting-things-a-lot-of-different-things/
//
// For example say given a case where:
// 	windowDuration is 20 minutes
// 	current window started 10 minutes ago
// 	requestCount would be 0.5 * currentWindowRequests + 0.5 * previousWindowRequests
func (rl *RateLimit) getInterpolatedRequestCount(key string) (requestCount int64) {
	now := time.Now()

	// calculate fraction of request that went in the current and previous windows
	currentWindowFraction := now.Sub(rl.currentWindow.startTime).Seconds() / rl.windowDuration().Seconds()
	previousWindowFraction := 1 - currentWindowFraction // thankfully this one's a bit easier to calculate!

	requestCount += int64(math.Round(
		float64(rl.currentWindow.getRequestCountFor(key)) *
			currentWindowFraction))
	requestCount += int64(math.Round(
		float64(rl.previousWindow.getRequestCountFor(key)) *
			previousWindowFraction))
	return
}

func (rl RateLimit) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	var key string
	if rl.isByHost() {
		// Separate remote IP and port; more lenient than net.SplitHostPort
		var ip string
		if idx := strings.LastIndex(r.RemoteAddr, ":"); idx > -1 {
			ip = r.RemoteAddr[:idx]
		} else {
			ip = r.RemoteAddr
		}
		key = ip
	} else {
		header := r.Header.Get(rl.ByHeader)
		if 0 == len(header) {
			// no header, no rate limit
			return next.ServeHTTP(w, r)
		}
		key = header
	}

	shouldBlock := rl.requestShouldBlock(key)

	if shouldBlock {
		fmt.Printf("Key %s exceeds rate limit for path %s.\n", key, r.URL.Path)
		w.WriteHeader(http.StatusTooManyRequests)
		if _, err := w.Write(nil); err != nil {
			return err
		}
	}
	return next.ServeHTTP(w, r)
}

var (
	_ caddy.Provisioner           = (*RateLimit)(nil)
	_ caddy.Validator             = (*RateLimit)(nil)
	_ caddyhttp.MiddlewareHandler = (*RateLimit)(nil)
	_ caddyfile.Unmarshaler       = (*RateLimit)(nil)
)
