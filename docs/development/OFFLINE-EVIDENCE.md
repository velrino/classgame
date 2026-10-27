# Recover unsent student work

The game saves each answer, reasoning response, and feedback response in IndexedDB **before** it advances the activity. Walking, reading, and leaving the board stay available while work is waiting to send.

## When work is waiting

Keep the same browser profile and site data. Reconnect and select **Retry**. The game sends the oldest response first and keeps its original event ID. A mission is complete only after every queued response is acknowledged and the API confirms completion. A partial send does not confirm completion.

If storage is blocked or full, the game stops accepting assessment input before changing the board. Allow site data and select Retry. If the attempt has reached a server limit, or the API rejects its evidence, the game keeps the work and asks for teacher help. Retry does not reset these permanent errors.

## Scope and storage

Each record belongs to one anonymous actor, class, mission, published revision, and attempt. Another actor or revision cannot replay it. A mission change, sign-out, local checkpoint reset, or failed send does not delete pending records. Older records remain separate; they are never attached to a new attempt.

IndexedDB stores the response, reasoning or feedback, published question fields needed for exact retry, internal scope IDs, and retry metadata. It does not store bearer tokens, student names, teacher observations, or upload files. This is browser storage, not encrypted private storage: scripts on the same origin can read it. Shared-device users should finish sending their work before leaving the browser.

The queue accepts at most 64 records per attempt, 256 records in total, 1 MiB of serialized records, and 16 KiB per new record. The transaction checks these limits across tabs. Capacity is a refusal to accept new input; it never evicts older responses. Browser quota errors also refuse new input. The browser may still evict site data, and users can clear it outside the app; IndexedDB is not a server backup.

## Retention and deletion

There is **no automatic expiry or TTL for unacknowledged work**. An acknowledged record is deleted immediately. Scope invalidation and ordinary recovery cleanup keep unacknowledged records. A future expiry policy needs an explicit product decision and user notice.

**Delete unsent work** is a separate, confirmed action. Its warning explains that work already sent stays with the teacher and the student may need to answer again. It deletes only this actor's pending records in this browser, clears the active local checkpoint, and returns to mission selection. It never deletes another actor's records or server evidence.

## Retry and recovery rules

IndexedDB transactions and a unique event index prevent whole-array lost updates. A short lease keeps two tabs from normally sending the same record together; durable server idempotency handles an uncertain response or an expired lease. Deleting an acknowledgment removes only its own event.

One send batch handles at most six records and lasts at most 60 seconds. One tutor request has a 45-second deadline. Storage transactions have a 10-second deadline. A record has at most five automatic failed attempts, with bounded backoff and numeric `Retry-After` support. Explicit Retry restarts transient retries. Navigation and scope changes abort the sender; an interrupted record remains pending. The render loop does not wait for network retries.

An older sessionStorage queue is imported only after the API confirms the exact attempt and revision and its source question IDs match the pinned mission. Its original copy stays until every import transaction succeeds. Unknown or invalid legacy work stays in place with a visible recovery error.

Production-policy evaluations cover scope, duplicate/conflicting event IDs, capacity refusal, retry policy, and the pending-completion gate. These checks do not prove real browser durability, cross-tab scheduling, storage eviction, or deployed access controls. Those need separate browser and release verification.
