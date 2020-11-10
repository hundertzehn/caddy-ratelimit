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
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("rate_limit", parseRateLimiter)
}

// parseCaddyfileHandler unmarshals tokens from h into a new middleware handler value.
//
// syntax example:
//     rate_limit {
//         by_header: string (eg. Authorization)
//         request_count: int,
//         time_frame: string (eg. 1h, 15m, 2d)
//     }
func parseRateLimiter(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var rl RateLimit
	err := rl.UnmarshalCaddyfile(h.Dispenser)
	return &rl, err
}