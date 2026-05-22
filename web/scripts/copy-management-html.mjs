import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = path.resolve(webRoot, '..');
const builtHtml = path.join(webRoot, 'dist', 'index.html');
const staticDir = path.join(repoRoot, 'static');
const targetHtml = path.join(staticDir, 'management.html');

if (!existsSync(builtHtml)) {
  throw new Error(`Frontend build output not found: ${builtHtml}`);
}

mkdirSync(staticDir, { recursive: true });
const html = readFileSync(builtHtml, 'utf8').replace(/\r\n?/g, '\n');
writeFileSync(targetHtml, html, 'utf8');
console.log(`Wrote ${path.relative(repoRoot, targetHtml)}`);
