---
worth: later
where: timeout.go:Timeout
added: 2026-11-28
---
# Timeout's header copy replaces Vary set outside it

`Timeout` buffers the inner handlers' headers in a fresh map and copies them back with
`maps.Copy(w.Header(), tw.h)`, which replaces any key set outside it. With `CORS` outside `Timeout` and
`Gzip` inside, Gzip's `Vary: Accept-Encoding` replaces CORS's `Vary: Origin`, so a shared cache can hand a
response stored without CORS headers to an allowed origin, which then fails its CORS check. PR #59 documents
the exception in the README CORS section. Surfaced reviewing PR #59.

Fix: merge `Vary` values in that copy instead of replacing them, with a CORS + Timeout + Gzip regression test
checking both `Origin` and `Accept-Encoding` survive. It changes Timeout's header semantics, so it goes in its
own PR.
