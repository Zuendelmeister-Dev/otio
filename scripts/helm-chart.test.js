'use strict';
const assert = require('node:assert/strict');
const {spawnSync} = require('node:child_process');
const path = require('node:path');
const fs = require('node:fs');
const root = path.resolve(__dirname, '..');
const helm = process.env.HELM_BIN || 'helm';
for (const name of ['sense.json', 'topology.json']) {
  const read = dir => JSON.parse(fs.readFileSync(path.join(root, dir, name), 'utf8'));
  assert.deepEqual(read('deploy/helm/otio/files'), read('deploy/kubernetes/config'), `Chart seed drift: ${name}`);
}
function run(args, success = true) {
  const result = spawnSync(helm, args, {cwd: root, encoding: 'utf8'});
  if (result.error) throw result.error;
  if (success) assert.equal(result.status, 0, result.stderr + result.stdout);
  else assert.notEqual(result.status, 0, 'Invalid values unexpectedly accepted');
  return result.stdout;
}
function render(extra = []) { return run(['template', 'plant-a', 'deploy/helm/otio', '--namespace', 'lab-a', ...extra]); }
function embeddedJSON(text, key) {
  const match = text.match(new RegExp('  '+key.replace('.', '\\.')+': \\|\\n((?:    .*\\n|\\n)+)'));
  assert(match, `Missing ${key}`);
  return JSON.parse(match[1].split('\n').map(line => line.slice(4)).join('\n'));
}
run(['lint', 'deploy/helm/otio', '--strict']);
const standard = render(['-f', 'examples/06-helm/values-kind.yaml']);
assert.equal((standard.match(/^kind: Deployment$/gm) || []).length, 6);
assert.equal((standard.match(/^kind: Service$/gm) || []).length, 6);
assert.equal((standard.match(/^kind: PersistentVolumeClaim$/gm) || []).length, 2);
assert.equal((standard.match(/helm.sh\/resource-policy: keep/g) || []).length, 2);
assert(!standard.includes('storageClassName:'), 'Default storage class must be omitted');
const config = embeddedJSON(standard, 'config.json');
const topology = embeddedJSON(standard, 'topology.json');
assert.equal(config.broker.host, 'plant-a-mqtt');
assert.equal(topology.senses[0].internalUrl, 'http://plant-a-sense:8100');
for (const source of config.sources) {
  const agent = topology.agents.find(a => a.agentId === source.agentId);
  assert.equal(agent.sourceHost, source.host);
  assert.equal(source.options.gatewayURL, 'http://plant-a-protocol-lab:8500');
  assert(new URL(source.options.connection.replace('modbus-rtu:tcp:', 'tcp:')).hostname.startsWith('plant-a-'));
}
const custom = render(['--set', 'persistence.retain=false', '--set-string', 'persistence.storageClass=fast', '--set-string', 'images.lense=registry.example/lense:42', '--set-string', 'credentials.rabbitmqPassword=a@b:c /d', '--set', 'imagePullSecrets[0].name=registry-auth']);
assert(custom.includes('storageClassName: "fast"'));
assert(custom.includes('image: "registry.example/lense:42"'));
assert(custom.includes('name: registry-auth'));
assert(!custom.includes('helm.sh/resource-policy: keep'));
const url = new URL(embeddedJSON(custom, 'config.json').sources.find(s => s.type === 'lab-amqp').options.connection);
assert.equal(decodeURIComponent(url.password), 'a@b:c /d');
assert.notEqual(standard.match(/checksum\/config: (\w+)/)[1], custom.match(/checksum\/config: (\w+)/)[1], 'Config changes must roll Pods');
assert(render(['--set-string', 'persistence.storageClass=']).includes('storageClassName: ""'));
run(['template', 'invalid', 'deploy/helm/otio', '--set', 'imagePullPolicy=invalid'], false);
run(['template', 'invalid', 'deploy/helm/otio', '--set-string', 'persistence.postgresSize=oops'], false);
console.log('Helm render, configuration, credential escaping, storage and validation tests passed');
