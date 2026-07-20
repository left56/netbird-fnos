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
| `POST /api/client/check-update` | 管理员查询指定官方 Release 的版本和 checksum |
| `POST /api/client/download` | 管理员下载、校验并安装指定官方 Release |
| `POST /api/client/upload` | 管理员上传 ELF 或官方 tar.gz 并验证安装 |
| `GET/POST/PUT/DELETE /api/download-sources` | 下载加速源读取及管理员管理 |
| `POST /api/download-sources/:id/test` | 管理员测试 HTTPS 加速源 |
| `POST /api/connect`, `POST /api/disconnect` | 管理员连接或断开官方 CLI；连接可一次性提交 `managementURL` 与 `setupKey` |
| `GET/POST/PUT/DELETE /api/profiles` | Profiles 查询及管理员管理 |
| `GET /api/networks` | 官方 CLI Networks 列表 |
| `POST /api/networks/select`, `POST /api/networks/deselect` | 管理员选择 Networks / Exit Node 路由 |
| `GET /api/logs/latest` | 最近 100 行经敏感字段脱敏的包装器与 daemon 日志 |

写入端点要求 fnOS 网关的 `X-Trim-Isadmin: true`（或 `1`）Header；读取端点对普通用户可用。下载仅接受官方 NetBird Release 和官方 checksum；加速源只能 URL-encode 完整官方 URL，且 TLS 校验不能关闭。上传不接受可执行路径或安装脚本。`setupKey` 仅作为本次官方 CLI 调用的参数，不会出现在响应、日志或诊断输出中。

When NetBird is not installed or cannot provide structured status, `/api/status` returns `state: "unavailable"` and `connected: false`. It never fabricates a connected state.
