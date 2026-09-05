import { defineConfig } from "vite";
import wails from "@wailsio/runtime/plugins/vite";

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  build: {
    // dist 还包含提交进仓库的 Host 回退页和 native-ui 嵌入资源，不能被 Vite 清空。
    emptyOutDir: false,
  },
  plugins: [wails("./bindings")],
});
