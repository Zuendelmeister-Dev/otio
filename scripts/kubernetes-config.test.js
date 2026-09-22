'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const root = path.resolve(__dirname, '..');
const json = p => JSON.parse(fs.readFileSync(path.join(root, p), 'utf8'));
const seed = json('deploy/kubernetes/config/sense.json');
assert.deepEqual(seed, json('examples/03-protocol-lab/config/config.json'), 'Kubernetes seed drifted from the tested Protocol Lab example');
const topology = json('deploy/kubernetes/config/topology.json');
const sense = topology.senses.find(s => s.id === seed.broker.clientId);
assert(sense, 'Sense MQTT identity must match the topology');
assert.equal(sense.internalUrl, 'http://sense:8100');
assert.equal(topology.presenses[0].internalUrl, 'http://protocol-lab:8500');
assert.equal(topology.agents.length, seed.sources.length);
for (const source of seed.sources) {
  const agent = topology.agents.find(a => a.agentId === source.agentId);
  assert(agent, `Missing topology for ${source.agentId}`);
  assert.equal(agent.senseId, sense.id);
  assert.equal(agent.sourceType, source.type);
  assert.equal(agent.sourceHost, source.host);
  assert(!source.options.connection.includes('localhost'), 'Collector must use cluster Service names');
}
console.log('Kubernetes source/topology consistency tests passed');
