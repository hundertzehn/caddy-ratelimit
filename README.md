# caddy-ratelimit plugin

Sample configuration in Caddyfile:

```
   rate_limit /api/* {
      by_header Authorization
      max_requests 180
      window_length 500
   }
```
