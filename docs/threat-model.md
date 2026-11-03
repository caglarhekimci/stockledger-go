# Threat model

API keys map to a tenant server-side; handlers derive tenant identity only from the authenticated key, never request JSON. Bodies are capped at 1 MiB. SQL uses parameters. Idempotency hashes bind a key to an item and quantity, rejecting altered retries. Remaining work for a production deployment includes a secret manager and rotation, TLS/reverse proxy, audit logging, per-key revocation, network policy, alerting, and load testing.
