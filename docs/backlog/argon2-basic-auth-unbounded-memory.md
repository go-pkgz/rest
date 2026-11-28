---
worth: later
where: basic_auth.go:BasicAuthWithArgon2Hash
added: 2026-11-28
---
# hashed BasicAuth has no bound on concurrent hash work

`BasicAuthWithArgon2Hash` runs argon2id with 64 MiB and 4 threads for every attempt, and bcrypt runs for
every attempt in the bcrypt variants, with no concurrency limit. A burst of junk Basic-auth requests can
exhaust a small service's memory. On master this already held for the configured user name ("admin" in the
README examples); PR #59 makes every user name pay it, which is the cost of closing the user-name timing
leak. Surfaced reviewing PR #59.

Fix: one README line under BasicAuth recommending `rest.Throttle(n)` in front of the hashed variants,
sized to the memory the service can spend on concurrent verifications. No code change.
