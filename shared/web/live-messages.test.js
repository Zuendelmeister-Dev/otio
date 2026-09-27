'use strict';
const assert = require('node:assert/strict');
const {messageView} = require('./live-messages.js');
const items = [
  {topic:'plant/A/temperature',payload:'<script>literal payload</script>'},
  {topic:'plant/A/status',payload:'ready'},
  {topic:'plant/A/temperature',payload:'24'},
  {topic:'plant/B/status',payload:'ready'},
];
const original = JSON.stringify(items);
assert.deepEqual(messageView(items).topics, ['plant/A/status','plant/A/temperature','plant/B/status']);
assert.deepEqual(messageView(items,'temperature').messages, [items[0],items[2]]);
assert.deepEqual(messageView(items,'status','plant/B/status').messages, [items[3]]);
assert.deepEqual(messageView(items,'temperature','plant/B/status').messages, []);
assert.deepEqual(messageView(items,'TEMPERATURE').messages, [], 'MQTT topics are case-sensitive');
assert.equal(messageView(items,'','gone/topic').missingSelection, true);
assert.deepEqual(messageView([], '', 'plant/A/status').messages, []);
assert.deepEqual(messageView(items,'  temperature  ').messages, [items[0],items[2]]);
assert(messageView([{topic:'new/topic'}], '', 'plant/A/status').missingSelection,
  'refresh must not silently reset a missing selection to All');
const burst = Array.from({length:60},(_,i)=>({topic:'topic/'+i}));
assert.equal(messageView(burst).recent.length,50);
assert.deepEqual(messageView(burst,'topic/59').messages,[], 'filter only the latest 50');
assert.equal(JSON.stringify(items),original,'filtering must not change the snapshot');
console.log('Live message bounds, topic search, selection and refresh semantics passed');
