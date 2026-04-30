/**
 * Vite 8 bundles with Rolldown; npm sometimes skips nested optional bindings (npm/cli#4828).
 * Install the matching @rolldown/binding-* for this OS/arch without modifying package.json.
 */
const { execSync } = require('node:child_process');
const { existsSync } = require('node:fs');
const path = require('node:path');

const ROOT = path.join(__dirname, '..');
const BINDING_VERSION = '1.0.0-rc.17';

const BINDINGS = [
  ['win32', 'x64', '@rolldown/binding-win32-x64-msvc'],
  ['darwin', 'arm64', '@rolldown/binding-darwin-arm64'],
  ['darwin', 'x64', '@rolldown/binding-darwin-x64'],
  ['linux', 'x64', '@rolldown/binding-linux-x64-gnu'],
  ['linux', 'arm64', '@rolldown/binding-linux-arm64-gnu'],
];

function main() {
  const row = BINDINGS.find(([p, a]) => p === process.platform && a === process.arch);
  if (!row) {
    console.warn(
      '[ensure-rolldown-binding] No mapped Rolldown binding for',
      process.platform,
      process.arch
    );
    return;
  }

  const name = row[2];
  const pkg = `${name}@${BINDING_VERSION}`;
  const installed = existsSync(path.join(ROOT, 'node_modules', name));

  if (installed) {
    return;
  }

  try {
    execSync(`npm install ${pkg} --no-save --no-fund --no-audit`, {
      cwd: ROOT,
      stdio: 'inherit',
    });
  } catch {
    console.warn('[ensure-rolldown-binding] Failed to install', pkg);
  }
}

main();
