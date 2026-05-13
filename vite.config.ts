import { defineConfig } from 'vite';
import { VitePluginNode } from 'vite-plugin-node';

export default defineConfig(({ mode }) => {
  return {
    server: {
      port: 3000,
    },
    plugins: [
      ...(mode !== 'test'
        ? VitePluginNode({
            adapter: 'telegraf',
            appPath: './src/index.ts',
            exportName: 'viteNodeApp',
            tsCompiler: 'esbuild',
          })
        : []),
    ],
    optimizeDeps: {
      exclude: ['telegraf'],
    },
  };
});
