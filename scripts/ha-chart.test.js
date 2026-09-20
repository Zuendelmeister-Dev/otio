'use strict';
const assert = require('node:assert/strict');
const {spawnSync} = require('node:child_process');
const helm = process.env.HELM_BIN || 'helm';
function run(args) {
  const r = spawnSync(helm, args, {encoding:'utf8'});
  if (r.error) throw r.error;
  assert.equal(r.status, 0, r.stderr + r.stdout);
  return r.stdout;
}
run(['lint','deploy/helm/otio-ha','--strict']);
const out = run(['template','ha','deploy/helm/otio-ha','-n','otio-ha']);
assert.equal((out.match(/^kind: Deployment$/gm)||[]).length, 2);
assert.equal((out.match(/^kind: Lease$/gm)||[]).length, 2);
assert.equal((out.match(/replicas: 2/g)||[]).length, 2);
assert.equal((out.match(/requiredDuringSchedulingIgnoredDuringExecution/g)||[]).length, 3);
assert(out.includes('instances: 3'));
assert(out.includes('additionalPlugins: [rabbitmq_mqtt]'));
assert(out.includes('dataDurability: required'));
assert(out.includes('OTIO_CONFIG_READ_ONLY'));
assert.equal((out.match(/otio.io\/active: "true"/g)||[]).length, 2);
assert(out.includes('maxUnavailable: 0, maxSurge: 1'));
assert(out.includes('minAvailable: 2'));
assert(out.includes('name: POD_NAME'));
assert(!out.includes('verbs: [get, list, watch, create, update]'));
assert(!out.includes('kind: ClusterRole'));
for (const key of ['config.json', 'topology.json']) {
  const pattern = new RegExp('  ' + key.replace('.', '\\.') + ': \\|\\n((?:    .*\\n|\\n)+)');
  const match = out.match(pattern);
  assert(match, key);
  const parsed = JSON.parse(match[1].split('\n').map(s => s.slice(4)).join('\n'));
  if (key === 'config.json') assert.equal(parsed.broker.host, 'ha-broker');
  else assert.equal(parsed.agents[0].senseId, 'ha-sense');
}
console.log('HA chart checks passed');
