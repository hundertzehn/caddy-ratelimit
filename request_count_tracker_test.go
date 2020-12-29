package ratelimit

import (
	"testing"
	"time"
)

func Test_RequestCountTracker_getRequestCountFor(t *testing.T) {
	hostName := "192.168.0.1"
	rct := RequestCountTracker{
		requestCount: map[string]int64{
			hostName: 200,
		},
		startTime: time.Time{},
		endTime:   time.Time{},
	}

	t.Run("Should append to existing host's counter", func(t *testing.T) {
		rct.addRequestFor(hostName)

		if rct.getRequestCountFor(hostName) != 201 {
			t.Errorf("Did not increment request count %+v", rct)
		}
	})

	t.Run("Should append to a new host's counter", func(t *testing.T) {
		newHostName := "10.0.0.127"
		rct.addRequestFor(newHostName)

		if rct.getRequestCountFor(newHostName) != 1 {
			t.Errorf("Did not insert/increment request count %+v", rct)
		}
	})

	t.Run("Should return 0 for unkown host's counter", func(t *testing.T) {
		unknownIPv6HostName := "2001:db8:85a3:8d3:1319:8a2e:370:7348"

		if rct.getRequestCountFor(unknownIPv6HostName) != 0 {
			t.Errorf("Should return 0 for unrecorded host %+v", rct)
		}
	})
}
