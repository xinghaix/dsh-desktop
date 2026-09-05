# debt 20260906
done: node helpers retarget + beta ABI align + lib rebuild
left: CI workflow scope

## Done this pass
- stock Node retarget in Host helpers (stable + beta)
- beta native-rebuild settings aligned (runtime=node, no electronjs.org headers)
- peer-node-version lib artifacts rebuilt; peer-electron-version leftovers removed
- beta tsdown entries aligned to Wails/Node Host surface
- update placeholders documented (prior)
- README launcher wording (prior)
- tests/scripts updated
## Pending
- product CI flip to smoke:wails / package:wails
- token needs workflow scope to push workflow files
- keep current download endpoints until a real Wails channel exists
- stale Electron main.ts in beta still imports deleted electron-runtime (quarantined from tsdown; not entry)

Notes:
- no .github/workflows changes this pass
- CI flip still needs workflow scope
