package ratelimit

import (
	"sync"
	"time"
)

// RequestCountTracker mixes by header and by host on the same structure
type RequestCountTracker struct {
	requestCount map[string]int64 // If 9,223,372,036,854,775,807 requests isn't enough...
	startTime    time.Time
	endTime      time.Time
	_mutex       *sync.RWMutex
}

// newRequestCountTracker returns a pointer to a blank initialised RequestCountTracker
func newRequestCountTracker(windowLength time.Duration) *RequestCountTracker {
	return &RequestCountTracker{
		requestCount: map[string]int64{},
		startTime:    time.Now(),
		endTime:      time.Now().Add(windowLength),
		_mutex:       &sync.RWMutex{},
	}
}

// newPreviousRequestCountTracker returns a pointer to a blank initialised RequestCountTracker for the
// previous windowLength, it's necessary for initial configuration
func newPreviousRequestCountTracker(windowLength time.Duration) *RequestCountTracker {
	return &RequestCountTracker{
		requestCount: map[string]int64{},
		startTime:    time.Now().Add(-windowLength),
		endTime:      time.Now(),
		_mutex:       &sync.RWMutex{},
	}
}

// addRequestFor adds to the request counter for specified key
func (rct *RequestCountTracker) addRequestFor(key string) {
	rct._mutex.Lock()
	rct.requestCount[key]++
	rct._mutex.Unlock()
}

// getRequestCounterForHost gets the request count for a given key
func (rct *RequestCountTracker) getRequestCountFor(key string) (requestCount int64) {
	rct._mutex.RLock()
	defer rct._mutex.RUnlock()
	return rct.requestCount[key]
}
