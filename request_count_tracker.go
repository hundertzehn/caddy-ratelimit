package ratelimit

import (
	"time"
)

// It mixes by header and by host on the same structure
type RequestCountTracker struct {
	requestCount map[string]int64 // If 9,223,372,036,854,775,807 requests isn't enough...
	startTime    time.Time
	endTime      time.Time
}

// newRequestCountTracker returns a pointer to a blank initialised RequestCountTracker
func newRequestCountTracker(windowLength time.Duration) *RequestCountTracker {
	return &RequestCountTracker{
		requestCount: map[string]int64{},
		startTime:    time.Now(),
		endTime:      time.Now().Add(windowLength),
	}
}

// addRequestFor adds to the request counter for specified key
func (rct *RequestCountTracker) addRequestFor(key string) {
	rct.requestCount[key] += 1
}

// getRequestCounterForHost gets the request count for a given key
func (rct RequestCountTracker) getRequestCountFor(key string) (requestCount int64) {
	return rct.requestCount[key]
}