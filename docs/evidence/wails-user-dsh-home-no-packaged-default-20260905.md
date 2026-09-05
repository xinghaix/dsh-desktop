# Acceptance delta: user-only Host default (2026-09-05)

Timezone: UTC logs; user Asia/Shanghai (UTC+8).

Branch: feat/wails-user-dsh-home

| # | Check | Result |
|---|-------|--------|
| 1 | Empty home: no monorepo host spawn | PASS (TestUserOnlyDefaultRejectsMonorepo) |
| 2 | User home Host preferred over packaged | PASS (TestUserHomeBeatsPackagedByDefault) |
| 3 | Packaged only with DSH_ALLOW_PACKAGED_HOST=1 | PASS (TestPackagedOnlyWhenAllowFlag) |
| 4 | Profile defaults to web | PASS (TestResolveActiveProfileDefaultsToWeb) |
| 5 | Chosen dir persisted | PASS (TestUserChosenDshHomePersisted) |
| 6 | go test ./wails | PASS |
| 7 | vitest cli-home-resolve | PASS (6) |
| 8 | host-install.html embedded | PASS (asset smoke) |

See wails-user-dsh-home-plugin-boundary-20260905.md
