# Local API

All endpoints return `{"status":"ok","data":...}` and are served through the fnOS unified gateway at `/app/netbird-fnos`.

| Endpoint | Data |
| --- | --- |
| `GET /api/health` | API service name |
| `GET /api/version` | embedded version, commit, and build time |
| `GET /api/status` | `state`, `connected`, and optional safe `detail` from the official CLI adapter |

When NetBird is not installed or cannot provide structured status, `/api/status` returns `state: "unavailable"` and `connected: false`. It never fabricates a connected state.
