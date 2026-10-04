// catalog.json is canonical; the browser bundle is checked in for static and
// Docker deployments. Run this script after changing metadata, or --check in CI.
'use strict';
const fs = require('node:fs');
const path = require('node:path');
const root = path.resolve(__dirname, '..');
const catalog = JSON.parse(fs.readFileSync(path.join(root, 'shared/protocolmeta/catalog.json'), 'utf8'));
const output = path.join(root, 'shared/web/protocol-capabilities.js');
const bundle = '// Generated from shared/protocolmeta/catalog.json; run scripts/generate-protocol-capabilities.cjs to update.\n' +
  '(function(root){\n  \'use strict\';\n  const catalog = ' + JSON.stringify(catalog, null, 2) + ';\n' +
  '  const sources = Object.fromEntries(catalog.flatMap(item => item.sources || []).map(item => [item.type, item]));\n' +
  '  const api = {catalog, sources};\n  root.IoTProtocolCapabilities = api;\n' +
  '  if(typeof module === \'object\' && module.exports) module.exports = api;\n' +
  '})(typeof window === \'undefined\' ? globalThis : window);\n';
const docsPath = path.join(root, 'docs/sense.md');
const startMarker = '<!-- protocol-capabilities:start -->';
const endMarker = '<!-- protocol-capabilities:end -->';
const docs = fs.readFileSync(docsPath, 'utf8');
const start = docs.indexOf(startMarker);
const end = docs.indexOf(endMarker);
if (start < 0 || end < start) throw new Error('Sense documentation is missing protocol capability markers.');
const table = [startMarker, '<!-- Generated from shared/protocolmeta/catalog.json. -->',
  '| Source type | Supported read modes | Protocol / implementation |', '|---|---|---|',
  ...catalog.flatMap(item => (item.sources || []).map(source =>
    '| `' + source.type + '` | ' + [source.poll ? '`poll`' : '', source.subscription ? '`subscription`' : ''].filter(Boolean).join(', ') +
    ' | ' + source.label.replace(/\|/g, '\\|') + ' (' + item.status + '). |')),
  endMarker].join('\n');
const generatedDocs = docs.slice(0, start) + table + docs.slice(end + endMarker.length);
if (process.argv.includes('--check')) {
  if (!fs.existsSync(output) || fs.readFileSync(output, 'utf8') !== bundle || docs !== generatedDocs) {
    console.error('Protocol capabilities assets or documentation are stale. Run node scripts/generate-protocol-capabilities.cjs.');
    process.exitCode = 1;
  } else console.log('Protocol capabilities assets and documentation match the canonical catalog.');
} else {
  fs.writeFileSync(output, bundle);
  fs.writeFileSync(docsPath, generatedDocs);
}
