Electron shell fallback (quarantined).

Primary: Wails + Node Host (start:wails / start:host).
Last resort: DSH_HOST_LAUNCHER=electron-main + DSH_ALLOW_ELECTRON_MAIN=1.
Do not mass-delete src/main.ts or electron-*.ts; electron-builder CI needs them.
