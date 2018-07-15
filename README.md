## REST helpers and middleware [![Build Status](https://travis-ci.org/go-pkgz/rest.svg?branch=master)](https://travis-ci.org/go-pkgz/rest) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/rest/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/rest?branch=master)

### AppInfo middleware

Adds info to every response header:
- App-Name - application name
- App-Version - application version
- Org - organization
- M-Host - host name from instance-level `$MHOST` env

### Ping-Pong middleware

Responds with `pong` on `GET /ping`. Also responds to anything with `/ping` suffix, like `/v2/ping` 

example for both:

```
> http GET https://remark42.radio-t.com/ping

HTTP/1.1 200 OK
Date: Sun, 15 Jul 2018 19:40:31 GMT
Content-Type: text/plain
Content-Length: 4
Connection: keep-alive
App-Name: remark42
App-Version: master-ed92a0b-20180630-15:59:56
Org: Umputun

pong
```

### Logger middleware

Logs all info about request, including user, method, status code, response size, url, elapsed time, request body (optional).
Can be customized by passing flags - LogNone, LogAll, LogUser and LogBody. Flags can be combined (provided multiple times)

### Recoverer middleware

Recoverer is a middleware that recovers from panics, logs the panic (and a backtrace), 
and returns a HTTP 500 (Internal Server Error) status if possible.

### Helpers

- `rest.JSON` - map alias, just for convenience `type JSON map[string]interface{}`
- `rest.RenderJSONFromBytes` - renders json response from []byte
- `rest.RenderJSONWithHTML` -  renders json response with html tags and forced `charset=utf-8`
- `rest.SendErrorJSON` - makes `{error: blah, details: blah}` json body and responds with given error code. Also adds context to logged message
