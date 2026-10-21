#!/usr/bin/env node
/**
 * Tìm các khóa i18n được dùng trong code (t('...')) nhưng thiếu trong messages/vi/*.json.
 * Chỉ bắt khóa tĩnh: `const t = useTranslations('ns')` / `getTranslations('ns')`,
 * biến bọc `const tl = looseT(t)`, rồi các lời gọi `t('key')`, `t.rich('key')`, `t.raw('key')`.
 * Chạy: node scripts/check-i18n.mjs   (exit 1 nếu thiếu khóa; --json để in dạng JSON)
 */
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

const root = new URL('..', import.meta.url).pathname;
const msgDir = join(root, 'messages', 'vi');
const messages = {};
for (const f of readdirSync(msgDir)) {
  if (f.endsWith('.json'))
    messages[f.slice(0, -5)] = JSON.parse(readFileSync(join(msgDir, f), 'utf8'));
}

function has(path) {
  let cur = messages;
  for (const part of path.split('.')) {
    if (cur == null || typeof cur !== 'object' || !(part in cur)) return false;
    cur = cur[part];
  }
  return true;
}

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (/\.(tsx?|jsx?)$/.test(name) && !p.includes('__tests__')) out.push(p);
  }
  return out;
}

const missing = new Map();
for (const file of walk(join(root, 'src'))) {
  const src = readFileSync(file, 'utf8');
  // Mỗi tên biến có thể khai báo nhiều lần (nhiều hàm trong 1 file) → lời gọi dùng khai báo gần nhất phía trên.
  const decls = [];
  const declRe =
    /(?:const|let)\s+(\w+)\s*=\s*(?:await\s+)?(?:useTranslations|getTranslations)\(\s*(?:'([^']*)'|"([^"]*)"|\{\s*namespace:\s*'([^']*)'[^}]*\})?\s*\)/g;
  for (const m of src.matchAll(declRe))
    decls.push({ name: m[1], ns: m[2] ?? m[3] ?? m[4] ?? '', at: m.index });
  const wrapRe = /(?:const|let)\s+(\w+)\s*=\s*looseT\(\s*(\w+)\s*\)/g;
  for (const m of src.matchAll(wrapRe)) {
    const base = decls.filter(d => d.name === m[2] && d.at < m.index).at(-1);
    if (base) decls.push({ name: m[1], ns: base.ns, at: m.index });
  }
  for (const name of new Set(decls.map(d => d.name))) {
    const callRe = new RegExp(
      `(?<![\\w.])${name}(?:\\.(?:rich|raw|markup))?\\(\\s*(?:'([^']+)'|"([^"]+)"|\`([^\`$]+)\`)`,
      'g',
    );
    for (const m of src.matchAll(callRe)) {
      const decl = decls.filter(d => d.name === name && d.at < m.index).at(-1);
      if (!decl) continue;
      const key = m[1] ?? m[2] ?? m[3];
      const full = decl.ns ? `${decl.ns}.${key}` : key;
      if (!has(full)) {
        if (!missing.has(full)) missing.set(full, new Set());
        missing.get(full).add(relative(root, file));
      }
    }
  }
}

const keys = [...missing.keys()].sort();
if (process.argv.includes('--json')) {
  console.log(JSON.stringify(Object.fromEntries(keys.map(k => [k, [...missing.get(k)]])), null, 2));
} else {
  for (const k of keys) console.log(`${k}  ←  ${[...missing.get(k)].join(', ')}`);
  console.log(keys.length ? `\n${keys.length} khóa thiếu.` : 'Không thiếu khóa i18n nào.');
}
process.exit(keys.length ? 1 : 0);
