// Copyright 2015 Matthew Holt and The Caddy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ratelimit

import (
	"fmt"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"strconv"
)

func parseRateLimit(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var rl RateLimit
	err := rl.UnmarshalCaddyfile(h.Dispenser)
	return rl, err
}

func (rl *RateLimit) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "by_header":
				d.NextArg()
				rl.ByHeader = d.Val()
			case "max_requests":
				d.NextArg()
				if num, err := strconv.Atoi(d.Val()); err != nil {
					return fmt.Errorf("max requests unit %v could not be parsed as a number", d.Val())
				} else {
					rl.MaxRequests = int64(num)
				}
			case "window_length":
				d.NextArg()
				if num, err := strconv.Atoi(d.Val()); err != nil {
					return fmt.Errorf("window_length unit %v could not be parsed as a number", d.Val())
				} else {
					rl.WindowLength = int64(num)
				}
			default:
				return d.Errf("unrecognized servers option '%s'", d.Val())
			}
		}
	}
	return nil
}

// Validate validates that the module has a usable config.
func (rl RateLimit) Validate() error {
	if rl.MaxRequests <= 0 || rl.WindowLength <= 0 {
		return fmt.Errorf("max_requests and window_length must be positive")
	}
	return nil
}
