'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const {catalog, sources} = require('../shared/web/protocol-capabilities.js');
const form = require('../shared/web/config-form.js');
assert.deepEqual(catalog, JSON.parse(fs.readFileSync(path.join(__dirname, '../shared/protocolmeta/catalog.json'), 'utf8')));
assert.equal(form.protocols, sources, 'the form must consume the shared capability source');
for (const item of catalog) {
  if (item.labRead) assert(sources['lab-' + item.id]?.requiresConnection, item.id + ' lacks a usable Sense source');
  if (!item.labRead) assert(!sources['lab-' + item.id], item.id + ' incorrectly advertises a native collector');
}
for (const [type, preset] of Object.entries(sources)) {
  const original = {agentId:'test',host:'device',readMode:'subscription',subscriptionIntervalMs:500,metrics:[{name:'value',register:42,nodeId:'old',address:'old',path:'old',scale:2,unit:''}]};
  const changed = form.changeProtocol(original, type);
  assert.equal(original.type, undefined, 'switching a draft must not mutate its previous source');
  assert.equal(changed.type, type);
  assert.equal(changed.port, preset.port);
  assert.equal(changed.readMode, preset.poll ? 'poll' : 'subscription');
  assert.deepEqual(Object.fromEntries(['register','nodeId','address','scale'].filter(key => key in changed.metrics[0]).map(key => [key,changed.metrics[0][key]])), preset.metric);
  assert(!('path' in changed.metrics[0]), 'old protocol addresses must not leak across protocol changes');
  if (!preset.subscription) assert(!('subscriptionIntervalMs' in changed));
  if (preset.requiresConnection) assert.equal(changed.options.connection, preset.connection);
}
assert.equal(form.changeProtocol({host:'meter',metrics:[{name:'temperature'}]}, 'lab-mbus-tcp').metrics[0].address, '1');
assert.throws(() => form.changeProtocol({}, 'unimplemented-protocol'), /Unknown protocol/);
for (const file of ['apps/sense/static/index.html','apps/lense/static/index.html','apps/dispense/static/index.html','apps/presense-modbus/main.go','apps/presense-opcua/main.go']) {
  const source = fs.readFileSync(path.join(__dirname,'..',file),'utf8');
  const catalogIndex = source.indexOf('/static/protocol-capabilities.js');
  assert(catalogIndex >= 0 && catalogIndex < source.indexOf('/static/config-form.js'), file + ' must load capabilities before the form');
}
console.log('Protocol catalog, form presets and asset loading checks passed.');
