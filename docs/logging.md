# Knocker Structured Logging

The Knocker service emits journald entries with a human-friendly `MESSAGE` and a consistent set of `KNOCKER_*` fields for machine consumption. This document describes the event contract so tooling (for example, a GNOME Shell extension) can parse the stream reliably.

## General Contract

- **Schema version:** All structured entries include `KNOCKER_SCHEMA_VERSION`. Current value: `"1"`.
- **Identifier:** Entries set `SYSLOG_IDENTIFIER=knocker` so they can be sliced from generic logs.
- **Event discriminator:** Every structured entry carries `KNOCKER_EVENT`, which determines the remaining fields.
- **Rendering:** Consume logs with `journalctl --user -u knocker.service -o json` (or `json-pretty`). Journald only supports explicit equality matches (`FIELD=value`) [per the manual](https://www.freedesktop.org/software/systemd/man/latest/journalctl.html), so to view every structured entry either:
  - pipe to a filter such as `jq 'select(.KNOCKER_EVENT != null)'`, or
  - specify an exact value, e.g. `journalctl --user -u knocker.service KNOCKER_EVENT=StatusSnapshot -o json`.
  Every `KNOCKER_*` value is encoded as a string because journald stores field payloads as strings.

Unless otherwise noted, fields may be absent when the corresponding value is unavailable. Consumers should treat missing fields as "unknown" rather than assuming an empty string.

## Event Catalogue

### `KNOCKER_EVENT=ServiceState`

Announces lifecycle transitions for the service.

| Field | Type | Description |
| --- | --- | --- |
| `KNOCKER_SERVICE_STATE` | enum | One of `"started"`, `"stopping"`, `"stopped"`, `"reloaded"` (reserved). |
| `KNOCKER_VERSION` | string (optional) | Knocker binary version, e.g. `"1.2.3"` or `"dev"`. |

Initialisation emits `started`. A graceful shutdown sequence raises `stopping` followed by `stopped`.

### `KNOCKER_EVENT=StatusSnapshot`

Provides the current state snapshot. Emitted at startup and whenever the snapshot changes materially.

| Field | Type | Description |
| --- | --- | --- |
| `KNOCKER_WHITELIST_IP` | string (optional) | Active whitelist IP (IPv4 or IPv6) if present. |
| `KNOCKER_WHITELIST_IPS_JSON` | JSON string (optional) | Reserved for future multi-IP responses (JSON array as a string). |
| `KNOCKER_EXPIRES_UNIX` | Unix timestamp (optional) | Expiry instant for the whitelist entry (seconds since epoch). |
| `KNOCKER_TTL_SEC` | integer string (optional) | TTL in seconds originally granted by the API. |
| `KNOCKER_NEXT_AT_UNIX` | Unix timestamp (optional) | Scheduled time for the next automatic knock. |
| `KNOCKER_CADENCE_SOURCE` | enum (optional) | Indicates whether the schedule comes from `ttl`, the API-provided `ttl_response`, or a configured `check_interval`. |
| `KNOCKER_PROFILE` | string (optional) | Reserved; profile name when multiple profiles are supported. |
| `KNOCKER_PORTS` | string (optional) | Comma-separated port list when known. |

### `KNOCKER_EVENT=WhitelistApplied`

Indicates the service (or CLI) applied a whitelist entry.

| Field | Type | Description |
| --- | --- | --- |
| `KNOCKER_WHITELIST_IP` | string | Whitelisted IP. |
| `KNOCKER_TTL_SEC` | integer string (optional) | TTL granted for the whitelist. |
| `KNOCKER_EXPIRES_UNIX` | Unix timestamp (optional) | Expiry instant, when provided by the API. |
| `KNOCKER_SOURCE` | enum (optional) | `"schedule"`, `"cli"`, or other future source identifiers. |
| `KNOCKER_PROFILE` | string (optional) | Reserved profile identifier. |

### `KNOCKER_EVENT=WhitelistExpired`

Signals that the currently tracked whitelist has expired or been cleared.

| Field | Type | Description |
| --- | --- | --- |
| `KNOCKER_WHITELIST_IP` | string (optional) | IP that expired. |
| `KNOCKER_EXPIRED_UNIX` | Unix timestamp (optional) | Time the entry expired. |

### `KNOCKER_EVENT=NextKnockUpdated`

Communicates a change to the scheduled next knock.

| Field | Type | Description |
| --- | --- | --- |
| `KNOCKER_NEXT_AT_UNIX` | Unix timestamp | Seconds since epoch for the next knock. "0" indicates the schedule is cleared. |
| `KNOCKER_CADENCE_SOURCE` | enum (optional) | Mirrors the cadence source reported in the snapshot. |
| `KNOCKER_PROFILE` | string (optional) | Reserved profile identifier. |
| `KNOCKER_PORTS` | string (optional) | Comma-separated port list when known. |

### `KNOCKER_EVENT=KnockTriggered`

Emitted whenever a knock attempt is triggered (manual or automatic).

| Field | Type | Description |
| --- | --- | --- |
| `KNOCKER_TRIGGER_SOURCE` | enum | `"schedule"`, `"cli"`, or `"external"` (reserved). |
| `KNOCKER_RESULT` | enum | `"success"` or `"failure"`. |
| `KNOCKER_WHITELIST_IP` | string (optional) | Whitelisted IP when the knock succeeds and returns one. |
| `KNOCKER_PROFILE` | string (optional) | Reserved profile identifier. |

Clients should watch for a matching `WhitelistApplied` event after a `success` result to update TTL and expiry.

### `KNOCKER_EVENT=Error`

Represents an operational error that should be surfaced to the user.

| Field | Type | Description |
| --- | --- | --- |
| `KNOCKER_ERROR_CODE` | enum | Machine-readable code (currently `"ip_lookup_failed"`, `"health_check_failed"`, `"knock_failed"`). |
| `KNOCKER_ERROR_MSG` | string | Human-readable context string. |
| `KNOCKER_CONTEXT` | string (optional) | Additional context (for example the IP or base URL involved). |

## Example Entry

```json
{
  "SYSLOG_IDENTIFIER": "knocker",
  "MESSAGE": "Whitelisted 1.2.3.4 for 600s (expires at 2025-06-14T10:01:40Z)",
  "PRIORITY": "6",
  "KNOCKER_SCHEMA_VERSION": "1",
  "KNOCKER_EVENT": "WhitelistApplied",
  "KNOCKER_WHITELIST_IP": "1.2.3.4",
  "KNOCKER_TTL_SEC": "600",
  "KNOCKER_EXPIRES_UNIX": "1750202500",
  "KNOCKER_SOURCE": "schedule"
}
```

## Parser Recommendations

- Treat field absence as "unknown". New fields may appear over time—ignore what you do not recognise.
- When multiple events arrive quickly, process `StatusSnapshot` last; it represents the new steady state after individual updates.
- Use the schema version to gate behaviour when future, incompatible changes are introduced.
- Prefer `KNOCKER_CADENCE_SOURCE` for determining whether the cadence is TTL-driven or configured via `check_interval`.

## Runtime Logs

In addition to structured journald entries, the service prints human-readable logs. After the cadence changes, look for messages such as:

- `Service running. Knocking every 9m0s (source: ttl).`
- `Service running. Checking for IP changes every 5m0s (source: check_interval).`
- `Adjusted knock cadence to 9m0s based on server TTL (540s).`

These logs mirror the cadence source reported in the structured events and help with manual diagnostics.
