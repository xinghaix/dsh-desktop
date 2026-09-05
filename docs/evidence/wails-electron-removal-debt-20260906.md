# debt 20260906
done: runtime shims to stock Node; nodeVersion renames; tests green
left: CI flip; Windows update URLs; Node ABI headers; docs sweep; workflow scope

## Done this pass
- desktop-runtime-environment: no RunAsNode=1 in shims; stock Node execPath
- renamed to nodeVersion / nodeExecutable across Host helpers+tests
- clear-env still strips inherited RunAsNode
- tests: desktop-runtime-environment 15 pass; desktop-terminal+recovery 14 pass

## Pending
- product CI still calls removed win/mac package scripts; prefer smoke:wails/package:wails
- win/mac runners: soft-skip until Go/Wails provisioned
- GitHub token needs workflow scope to push workflow files
- update-download URLs still dshdesktop.cn /api/downloads/windows and /mac
- drop old headers disturl; rebuild natives for Node 22 ABI
- docs: legacy-builder primacy + start:host:legacy-as-node mentions
