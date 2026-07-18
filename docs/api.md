# Local API

All endpoints return `{"status":"ok","data":...}` and are served through the fnOS unified gateway at `/app/netbird-fnos`.

| Endpoint | Data |
| --- | --- |
| `GET /api/health` | API service name |
| `GET /api/version` | embedded version, commit, and build time |
| `GET /api/status` | `state`, `connected`, and optional safe `detail` from the official CLI adapter |
| `GET /api/client` | 当前受控 CLI 的来源、版本、架构、checksum 和能力 |
| `GET /api/client/versions` | 已安装的托管版本 |
| `POST /api/client/switch` | 管理员切换指定托管版本 |
| `POST /api/client/rollback` | 管理员回滚到上一个版本 |
| `POST /api/client/use-bundled` | 管理员回退到 FPK 内置版本 |
| `DELETE /api/client/versions/:version` | 管理员删除非当前历史版本 |

写入端点要求 fnOS 网关的 `X-Trim-Isadmin: true`（或 `1`）Header；读取端点对普通用户可用。`check-update`、`download`、`upload` 和下载源配置端点已预留受控路由，但在本构建中返回 `501`，不会接收任意 URL、可执行文件路径或关闭 TLS 校验的请求。

When NetBird is not installed or cannot provide structured status, `/api/status` returns `state: "unavailable"` and `connected: false`. It never fabricates a connected state.
