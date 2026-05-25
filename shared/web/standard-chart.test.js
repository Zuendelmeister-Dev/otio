
const fs = require('fs');
const assert = require('assert');

const code = fs.readFileSync('shared/web/standard-chart.js', 'utf8');
const attachStart = code.indexOf('function attachTooltip()');
const attachEnd = code.indexOf('attachTooltip();', attachStart);
const attachCode = code.slice(attachStart, attachEnd);

assert(attachStart >= 0, 'expected attachTooltip');
assert(!attachCode.includes('setupCanvas('), 'tooltip must not reset the canvas during hover');
assert(attachCode.includes('mouseY'), 'tooltip must consider vertical mouse position');
assert(attachCode.includes('Math.sqrt(dx * dx + dy * dy)'), 'tooltip should use true 2D proximity');
assert(code.includes('function isShapePoint(points, index)'), 'chart should mark local extrema and spikes');
assert(code.includes('isImportantMarker ? radius + 0.8 : radius'), 'important metric markers should be emphasized');
assert(code.includes('state.hitPoints.push(item);'), 'displayed markers should be hover targets for quick view');

console.log('standard-chart quick metrics regression test passed');
