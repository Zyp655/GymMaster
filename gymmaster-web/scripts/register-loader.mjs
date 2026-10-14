import fs from 'node:fs';
import { registerHooks } from 'node:module';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const srcDir = path.join(rootDir, 'src');

function tryResolveTs(filePath) {
  for (const ext of ['.ts', '.tsx', '/index.ts', '/index.tsx']) {
    const candidate = filePath + ext;
    if (fs.existsSync(candidate) && fs.statSync(candidate).isFile()) {
      return pathToFileURL(candidate).href;
    }
  }
  return null;
}

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.startsWith('@/')) {
      const target = path.join(srcDir, specifier.slice(2));
      if (fs.existsSync(target) && fs.statSync(target).isFile()) {
        return nextResolve(pathToFileURL(target).href, context);
      }
      const resolved = tryResolveTs(target);
      if (resolved) return nextResolve(resolved, context);
    }
    if (
      (specifier.startsWith('./') || specifier.startsWith('../')) &&
      !path.extname(specifier) &&
      context.parentURL?.startsWith('file:')
    ) {
      const parentDir = path.dirname(fileURLToPath(context.parentURL));
      if (parentDir.startsWith(srcDir)) {
        const resolved = tryResolveTs(path.resolve(parentDir, specifier));
        if (resolved) return nextResolve(resolved, context);
      }
    }
    return nextResolve(specifier, context);
  },
});
