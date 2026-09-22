
const fs = require('fs');
const vm = require('vm');
const assert = require('assert');

const code = fs.readFileSync('shared/web/iot-ui.js', 'utf8');

function makeContext() {
  const storage = {};
  return {
    window: {
      IoTStandardChart: {
        escapeHtml: value => String(value),
        formatTimeLabel: value => value
      },
      devicePixelRatio: 1
    },
    localStorage: {
      getItem: key => storage[key] || null,
      setItem: (key, value) => { storage[key] = value; }
    },
    document: {
      getElementById: () => null
    },
    console
  };
}

const context = makeContext();
vm.createContext(context);
vm.runInContext(code, context);

assert.equal(typeof context.window.IoTUI.renderConnectionOverview, 'function');
assert.equal(typeof context.window.IoTUI.rememberBinaryStatus, 'function');

const points = context.window.IoTUI.rememberBinaryStatus(
  'test-component',
  true,
  '2026-05-20T12:00:00.000Z',
  1440
);

assert(points.length >= 1);
assert.equal(points[points.length - 1].value, true);
assert(points[0].timestamp.startsWith('2026-05-20'));
const remember = context.window.IoTUI.rememberBinaryStatus;
assert.equal(remember('test-component', true, '2026-05-20T12:00:05.000Z', 1440).length, 2, 'Unchanged states still need history samples');
assert.equal(remember('test-component', true, '2026-05-20T12:00:05.000Z', 1440).length, 2, 'Duplicate observations must not grow history');
const denied = makeContext();
denied.localStorage.getItem = () => { throw Error('storage denied'); };
denied.localStorage.setItem = () => { throw Error('storage denied'); };
vm.createContext(denied); vm.runInContext(code, denied);
const fallback = denied.window.IoTUI.rememberBinaryStatus;
fallback('offline', true, '2026-05-20T12:00:00Z', 1440);
assert.equal(fallback('offline', false, '2026-05-20T12:00:05Z', 1440).length, 2, 'Keep history in memory when browser storage is denied');
for(let i=0;i<310;i++) fallback('bounded', true, new Date(Date.UTC(2026,4,20,12,0,i)).toISOString(),1440);
assert.equal(fallback('bounded', false,'2026-05-20T12:06:00Z',1440).length,300);

console.log('iot-ui shared tests passed');


const uiCode = require('fs').readFileSync('shared/web/iot-ui.js', 'utf8');
assert(uiCode.includes('points[index - 1].value !== point.value'), 'transition markers should be emphasized');
assert(uiCode.includes('points[index + 1].value !== point.value'), 'transition markers should be hover targets');
