# Enrollment and access

Sign-in proves identity. It does not make every account a teacher.

## Teacher accounts

`AUTH_TEACHER_PROVIDERS` selects sign-in methods. `AUTH_TEACHER_EMAIL_ALLOWLIST` selects exact teacher email addresses. Production requires a nonempty list, and listed accounts must have verified email. Wildcards are rejected. Classroom ownership is still checked separately.

Development can leave the list empty for isolated setup. This is an explicit local policy, not a production invitation system. A rejected account receives `403 teacher_not_allowed`; no allowlist values are returned. Bootstrap is off by default and needs an explicit setting.

## Student joins

A six-digit class code is a public classroom identifier, not a password. New classes accept joins for 30 days. Teachers can close joins or set a future expiry within 90 days in the class form. Existing class records keep their prior behavior until the teacher sets a policy; this release does not rewrite shared records.

The policy is checked inside the membership transaction. Closing joins during a request cannot admit a new student using a stale read. Existing members keep access; closing joins is not a student-removal operation. Inactive classes reject all joins. A student moving to another tenant/class cannot reuse the old class memory.

Codes stay tied to stable classroom IDs. To replace a shared code, close joins and create a new class. There is no hidden code rotation that would break pinned attempts or old links.

Public code validation and authenticated joins use shared Redis abuse limits. Redis failure rejects admission; it does not remove pending browser evidence. Repeated guesses are still a reason to close joins and review access. A code must never grant teacher access.

## Client IP and proxy trust

An empty `TRUSTED_PROXY_CIDRS` ignores forwarded IP headers in every environment. Add only the actual trusted proxy CIDRs. Universal ranges are rejected. The full chain is checked from the direct peer; a direct untrusted client cannot choose its rate-limit key with `X-Forwarded-For`.

Do not copy a private-network range without checking the real load balancer path. Cloud hosting, firewall rules, proxy settings, and broker TLS/ACL/no-eviction remain explicit environment setup. No hosted service is created by these settings.

## Model and attachment boundaries

Model output is untrusted content, not authorization. Published questions, valid reference IDs, private teacher projections, and server evidence remain authoritative. Input/output/time/concurrency bounds and shared provider admission limit cost and contention. Human review remains required before publishing a fallback or generated mission.

The deterministic browser flow confirmed that fallback questions may be generic. Teachers must review whether a preview matches the requested lesson before approval. These local checks do not measure a live model's teaching quality or general prompt-injection resistance.

For object access use the [private access procedure](PRIVATE-ATTACHMENTS.md); for safe data recovery use the retention guide. Source checks do not deploy Firebase rules or change cloud access.
