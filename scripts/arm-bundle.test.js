'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {execFileSync} = require('node:child_process');
const root = path.resolve(__dirname,'..');
const vars = fs.readFileSync(path.join(root,'deploy/ansible/group_vars/all.yml'),'utf8');
function list(name) {
  const section = vars.match(new RegExp('^'+name+':\\s*\\n((?:[ \\t]+- [^\\n]+\\n?)+)','m'));
  assert(section, 'Missing Ansible list '+name);
  return [...section[1].matchAll(/- ([^\r\n]+)/g)].map(m=>m[1].trim());
}
const entries = [...list('otio_source_directories'),...list('otio_source_files')];
const tasks = fs.readFileSync(path.join(root,'deploy/ansible/roles/deploy_stack/tasks/main.yml'),'utf8');
assert(tasks.includes('otio_source_directories + otio_source_files'), 'Archive task must use the tested source lists');
assert(!/state: absent[\s\S]{0,100}- config/.test(tasks), 'Deploy must preserve runtime config');
const folder = fs.mkdtempSync(path.join(os.tmpdir(),'otio-arm-context-'));
try {
  const archive = path.join(folder,'source.tar.gz');
  execFileSync('tar',['-czf',archive,...entries],{cwd:root,stdio:'pipe'});
  const checkout = path.join(folder,'repo'); fs.mkdirSync(checkout);
  execFileSync('tar',['-xzf',archive,'-C',checkout],{stdio:'pipe'});
  const files = [];
  function walk(dir) { for (const entry of fs.readdirSync(dir,{withFileTypes:true})) { const full=path.join(dir,entry.name); if(entry.isDirectory())walk(full);else files.push(path.relative(checkout,full).replaceAll('\\','/')); } }
  walk(checkout);
  for (const dockerfile of files.filter(f=>f.startsWith('apps/') && f.endsWith('/Dockerfile'))) {
    const text=fs.readFileSync(path.join(checkout,dockerfile),'utf8');
    for (const match of text.matchAll(/^COPY\s+(?!-)(.+)\s+\S+\s*$/gm)) {
      for (const input of match[1].trim().split(/\s+/)) {
        const escaped=input.replace(/\/$/,'').replace(/[.+?^${}()|[\]\\]/g,'\\$&').replaceAll('*','[^/]*');
        const matcher=new RegExp('^'+escaped+'(?:/|$)');
        assert(files.some(file=>matcher.test(file)), `${dockerfile}: archive missing COPY input ${input}`);
      }
    }
  }
  for (const seed of ['apps/sense','apps/lense']) {
    assert(fs.existsSync(path.join(checkout,seed,'Dockerfile')));
    assert(fs.existsSync(path.join(checkout,'examples/03-protocol-lab/config/config.json')));
  }
  console.log('ARM source archive contains every application Docker COPY input');
} finally {
  // Only remove the exact fresh directory created by this test.
  assert(path.dirname(folder)===path.resolve(os.tmpdir()) && path.basename(folder).startsWith('otio-arm-context-'));
  fs.rmSync(folder,{recursive:true,force:true});
}
