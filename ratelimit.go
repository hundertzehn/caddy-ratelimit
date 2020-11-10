package ratelimit

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"github.com/caddyserver/caddy/v2"
)

func init() {
	caddy.RegisterModule(RateLimit{})
}

// rateLimitOptions stores options detailing how rate limiting should be applied,
// as well as the current and previous window's hosts:requestCount mapping
type RateLimit struct {

	// window length for request rate checking (>= 5 minutes)
	WindowLength string `json:"window_length"`

	// duration derived, from WindowLength
	windowDuration time.Duration

	// max request that should be processed per host in a given windowDuration
	MaxRequestsString string `json:"max_requests"`

	// max requests, derived from MaxRequestsString
	maxRequests int64

	// current window's request count per host
	currentWindow *RequestCountTracker

	// previous window's request count per host
	previousWindow *RequestCountTracker
}

// CaddyModule returns the Caddy module information.
func (RateLimit) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.ratelimit",
		New: func() caddy.Module { return new(RateLimit) },
	}
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (rl *RateLimit) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.Args(&rl.WindowLength) || !d.Args(&rl.MaxRequestsString) {
			// not enough args
			return d.ArgErr()
		}

		var durationUnit time.Duration
		// last character should be duration unit
		switch rune(rl.WindowLength[len(rl.WindowLength)-1]) {
		case 'd':
			durationUnit = 24 * time.Hour
		case 'h':
			durationUnit = time.Hour
		case 'm':
			durationUnit = time.Minute
		default:
			return fmt.Errorf("unknown duration unit %v, valid values are d,h,m", durationUnit)
		}

		// everything before last character should be an int64 for duration multiplier
		// trim space to allow formats: 4h and 4 h
		durationString := strings.TrimSpace(rl.WindowLength[:len(rl.WindowLength)-1])
		if num, err := strconv.Atoi(durationString); err != nil {
			return fmt.Errorf("duration unit %v could not be parsed as a number", durationString)
		} else {
			rl.windowDuration = time.Duration(num) * durationUnit
		}

		// parsing max request count for time period
		if num, err := strconv.Atoi(rl.MaxRequestsString); err != nil {
			return fmt.Errorf("request count %v could not be parsed as a number", rl.MaxRequestsString)
		} else {
			rl.maxRequests = int64(num)
		}

		if d.NextArg() {
			// too many args
			return d.ArgErr()
		}

		rl.setupRateLimit(rl.windowDuration, rl.maxRequests)
	}
	return nil
}

// setupRateLimit sets up the package-level variable `rateLimiter`,
// and starts the auto-window refresh process
func (rl *RateLimit) setupRateLimit(windowLength time.Duration, maxRequests int64) {
	rl.windowDuration = windowLength
	rl.maxRequests = maxRequests
	rl.currentWindow = newRequestCountTracker(windowLength)
	rl.previousWindow = &RequestCountTracker{}

	go func() { // automatic shuffling of request count tracking windows
		for {
			time.Sleep(rl.currentWindow.endTime.Sub(time.Now()))
			rl.refreshWindows()
		}
	}()

	return
}

// refreshWindows() checks if currentWindow has reached its expiry time, and if it has,
// moves currentWindow to previousWindow, and re-initialises currentWindow
func (rl *RateLimit) refreshWindows() (didRefresh bool) {
	if rl.currentWindow.endTime.Before(time.Now()) {
		rl.previousWindow = rl.currentWindow
		rl.currentWindow = newRequestCountTracker(rl.windowDuration)

		didRefresh = true
	}

	return
}

// requestShouldBlock checks whether the request from a given host name should block,
// and increments the request counter for the hostName first
// will block if current request would push the hostName over the blocking threshold
func (rl *RateLimit) requestShouldBlock(hostName string) (shouldBlock bool) {
	rl.currentWindow.addRequestForHost(hostName)                     // increment request counter for host
	return rl.getInterpolatedRequestCount(hostName) > rl.maxRequests // check if they now are above the request limit
}

// getInterpolatedRequestCount gets an interpolated request count for a specified hostName
// Always considers requests across the given windowDuration
// More details: https://blog.cloudflare.com/counting-things-a-lot-of-different-things/
//
// For example say given a case where:
// 	windowDuration is 20 minutes
// 	current window started 10 minutes ago
// 	requestCount would be 0.5 * currentWindowRequests + 0.5 * previousWindowRequests
func (rl RateLimit) getInterpolatedRequestCount(hostName string) (requestCount int64) {
	now := time.Now()

	// calculate fraction of request that went in the current and previous windows
	currentWindowFraction := now.Sub(rl.currentWindow.startTime).Seconds() / rl.windowDuration.Seconds()
	previousWindowFraction := 1 - currentWindowFraction // thankfully this one's a bit easier to calculate!

	requestCount += int64(math.Round(
		float64(rl.currentWindow.getRequestCountForHost(hostName)) *
			currentWindowFraction))
	requestCount += int64(math.Round(
		float64(rl.previousWindow.getRequestCountForHost(hostName)) *
			previousWindowFraction))

	return
}

func (rl *RateLimit) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Separate remote IP and port; more lenient than net.SplitHostPort
	var ip string
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx > -1 {
		ip = r.RemoteAddr[:idx]
	} else {
		ip = r.RemoteAddr
	}

	shouldBlock := rl.requestShouldBlock(ip)

	if shouldBlock {
		w.WriteHeader(http.StatusTooManyRequests)
		if _, err := w.Write(nil); err != nil {
			return err
		}
	}

	return next.ServeHTTP(w, r)
}