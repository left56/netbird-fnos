# Provisional fnOS package layout

`manifest/` is intentionally a TODO-only placeholder until fnOS metadata requirements are verified. The package payload produced by `make package` is:

```
app/bin/netbird-fnos-api
app/etc/netbird-fnos.env.example
app/systemd/netbird-fnos-api.service
app/var/
app/web/
scripts/{install,start,stop,upgrade,uninstall}.sh
manifest/
```

Lifecycle scripts are strict and idempotent. Uninstall retains the official NetBird identity data; `PURGE_NETBIRD_FNOS_STATE=1` removes only the wrapper state directory.
