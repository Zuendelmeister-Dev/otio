// Check the prospective Git contents without printing secret values.
'use strict';
const fs = require('node:fs');
const path = require('node:path');
const {execFileSync} = require('node:child_process');
const root = path.resolve(__dirname, '..');
const files = [...new Set(execFileSync('git', ['ls-files', '-z', '--cached', '--others', '--exclude-standard'],
  {cwd:root, encoding:'utf8'}).split('\0').filter(Boolean))];
const failures = [];
for (const file of files) {
  const full = path.join(root, file);
  if (!fs.existsSync(full) || !fs.statSync(full).isFile()) continue;
  if (/(^|\/)(node_modules|coverage|docker-data|certificates|certificate-profiles|certificates-local)\//.test(file) ||
      /\.(key|pem|p12|pfx|exe|db|sqlite3?|dump|tar)$/i.test(file) ||
      /(^|\/)\.env(?:$|\.(?!example$|sample$))/.test(file) ||
      /docs\/releases\/.*\/(PUBLISHING\.md|COMMIT_MESSAGE\.txt|PR_DESCRIPTION\.md)$/.test(file)) {
    failures.push(`${file}: local/runtime material is a Git candidate`);
  }
  if (fs.statSync(full).size > 10*1024*1024) failures.push(`${file}: exceeds 10 MiB; review before publishing`);
  if (!/\.(md|js|cjs|go|json|ya?ml|ps1|sh|txt)$/.test(file)) continue;
  const text = fs.readFileSync(full, 'utf8');
  if (/^-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/m.test(text) ||
      /(?:ghp_|github_pat_)[A-Za-z0-9_]{30,}/.test(text)) failures.push(`${file}: possible embedded credential`);
  if (!file.endsWith('.md')) continue;
  // Validate local file/image links; headings and external links are outside this check.
  const prose = text.replace(/```[\s\S]*?```/g, '');
  for (const match of prose.matchAll(/!?\[[^\]]*\]\(([^\s)]+)(?:\s+"[^"]*")?\)/g)) {
    const link = match[1];
    if (/^(?:[a-z]+:|#|\/)/i.test(link)) continue;
    const target = decodeURIComponent(link.split('#')[0]);
    if (target && !fs.existsSync(path.resolve(path.dirname(full),target))) failures.push(`${file}: missing link ${target}`);
  }
}
if (failures.length) {console.error(failures.join('\n'));process.exitCode=1;}
else console.log(`Repository hygiene and local documentation links passed (${files.length} candidate files).`);
