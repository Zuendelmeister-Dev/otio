
(function () {
  function escapeHtml(value) {
    return window.IoTStandardChart && window.IoTStandardChart.escapeHtml
      ? window.IoTStandardChart.escapeHtml(value)
      : String(value)
          .replaceAll('&', '&amp;')
          .replaceAll('<', '&lt;')
          .replaceAll('>', '&gt;')
          .replaceAll('"', '&quot;')
          .replaceAll("'", '&#039;');
  }

  function statusDot(ok, warning) {
    return '<span class="dot ' + (ok ? 'ok' : warning ? 'warn' : '') + '"></span>';
  }

  function renderStatusDetails(elementOrId, items) {
    const element = typeof elementOrId === 'string' ? document.getElementById(elementOrId) : elementOrId;
    if (!element) return;
    element.innerHTML = (items || []).map(item => {
      const ok = Boolean(item.ok);
      const warning = Boolean(item.warning);
      return '<div class="health-item" title="' + escapeHtml(item.label + ': ' + item.value) + '">' +
        '<span>' + statusDot(ok, warning) + escapeHtml(item.label) + '</span>' +
        '<span>' + escapeHtml(item.value) + '</span>' +
        '</div>';
    }).join('');
  }

  function renderOverviewCards(elementOrId, items) {
    const element = typeof elementOrId === 'string' ? document.getElementById(elementOrId) : elementOrId;
    if (!element) return;
    element.innerHTML = '<div class="overview-grid">' + (items || []).map(item => {
      return '<div class="overview-card">' +
        '<div class="k">' + escapeHtml(item.label) + '</div>' +
        '<div class="overview-value">' + escapeHtml(item.value) + '</div>' +
        (item.hint ? '<div class="overview-hint">' + escapeHtml(item.hint) + '</div>' : '') +
        '</div>';
    }).join('') + '</div>';
  }

  function renderGroupedLogs(elementOrId, logs, selectedLevels) {
    const element = typeof elementOrId === 'string' ? document.getElementById(elementOrId) : elementOrId;
    if (!element) return;
    const selected = selectedLevels || new Set(['INFO', 'WARN', 'ERROR']);
    const grouped = {};
    (logs || []).forEach(item => {
      const level = String(item.level || item.severity || 'INFO').toUpperCase();
      if (!selected.has(level)) return;
      const component = item.component || item.agentId || 'system';
      const message = item.message || '';
      const timestamp = item.timestamp || item.ts || item.lastSeen || '';
      const firstSeen = item.firstSeen || timestamp;
      const lastSeen = item.lastSeen || timestamp;
      const itemCount = Number(item.count || 1);
      const key = level + '|' + component + '|' + message;
      if (!grouped[key]) {
        grouped[key] = { level, component, message, count: 0, firstSeen, lastSeen };
      }
      grouped[key].count += itemCount;
      if (String(lastSeen) > String(grouped[key].lastSeen)) grouped[key].lastSeen = lastSeen;
      if (String(firstSeen) < String(grouped[key].firstSeen)) grouped[key].firstSeen = firstSeen;
    });

    const rows = Object.values(grouped).sort((a, b) => String(b.lastSeen).localeCompare(String(a.lastSeen)));
    if (!rows.length) {
      element.innerHTML = '<div class="k">No logs for the selected filter.</div>';
      return;
    }

    element.innerHTML = rows.slice(0, 250).map(item => {
      const levelClass = item.level === 'ERROR' ? 'log-error' : item.level === 'WARN' ? 'log-warn' : 'log-info';
      const count = item.count > 1 ? ' <span class="status-pill">x' + item.count + '</span>' : '';
      return '<pre><span class="' + levelClass + '">' + statusDot(item.level === 'INFO', item.level === 'WARN') + item.level + '</span>' +
        count + ' [' + escapeHtml(item.lastSeen) + '] ' + escapeHtml(item.component) + ': ' + escapeHtml(item.message) +
        '<br><span class="k">first seen: ' + escapeHtml(item.firstSeen) + '</span></pre>';
    }).join('');
  }

  function renderBinaryStatusChart(options) {
    const canvas = document.getElementById(options.canvasId);
    const legend = options.legendId ? document.getElementById(options.legendId) : null;
    const tooltip = options.tooltipId ? document.getElementById(options.tooltipId) : null;
    if (!canvas) return;

    const ratio = window.devicePixelRatio || 1;
    const width = canvas.clientWidth || canvas.width || 900;
    const height = options.height || canvas.clientHeight || 260;
    canvas.width = Math.floor(width * ratio);
    canvas.height = Math.floor(height * ratio);
    const ctx = canvas.getContext('2d');
    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);

    const points = (options.points || [])
      .map(point => ({ timestamp: point.timestamp, value: Boolean(point.value) }))
      .filter(point => Number.isFinite(new Date(point.timestamp).getTime()))
      .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());

    const label = options.label || 'connected';
    const color = options.color || '#ff8a1d';
    const left = 120, right = 30, top = 24, bottom = 46;
    const plotWidth = width - left - right;
    const connectedY = top + 28;
    const disconnectedY = height - bottom - 28;
    const times = points.map(p => new Date(p.timestamp).getTime());
    const nowForDomain = Date.now();
    const minTime = options.domainStart ? new Date(options.domainStart).getTime() : (options.fitToData ? (times.length ? Math.min(...times) : nowForDomain - 24 * 60 * 60 * 1000) : nowForDomain - 24 * 60 * 60 * 1000);
    const maxTime = options.domainEnd ? new Date(options.domainEnd).getTime() : (options.fitToData ? (times.length ? Math.max(...times) : nowForDomain) : nowForDomain);
    const getX = timestamp => {
      const t = new Date(timestamp).getTime();
      return left + ((t - minTime) / Math.max(1, maxTime - minTime)) * plotWidth;
    };
    const getY = value => value ? connectedY : disconnectedY;

    function drawBase() {
      ctx.clearRect(0, 0, width, height);
      ctx.strokeStyle = '#2a2f38';
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(left, top);
      ctx.lineTo(left, height - bottom);
      ctx.lineTo(width - right, height - bottom);
      ctx.stroke();

      ctx.font = '12px Inter, system-ui, sans-serif';
      ctx.fillStyle = '#8a9099';
      ctx.fillText('Connected', 16, connectedY + 4);
      ctx.fillText('Disconnected', 16, disconnectedY + 4);

      ctx.strokeStyle = 'rgba(42,47,56,.75)';
      [connectedY, disconnectedY].forEach(y => {
        ctx.beginPath();
        ctx.moveTo(left, y);
        ctx.lineTo(width - right, y);
        ctx.stroke();
      });

      [minTime, (minTime + maxTime) / 2, maxTime].forEach((t, index) => {
        const x = left + (plotWidth / 2) * index;
        ctx.strokeStyle = 'rgba(42,47,56,.5)';
        ctx.beginPath();
        ctx.moveTo(x, top);
        ctx.lineTo(x, height - bottom);
        ctx.stroke();
        ctx.fillStyle = '#8a9099';
        ctx.fillText((window.IoTStandardChart ? window.IoTStandardChart.formatTimeLabel(new Date(t).toISOString()) : new Date(t).toLocaleTimeString()), x - 28, height - 18);
      });
    }

    function drawData(highlight) {
      if (!points.length) {
        ctx.fillStyle = '#8a9099';
        ctx.fillText('No status samples available.', left + 16, top + 24);
        if (legend) legend.innerHTML = '';
        return;
      }

      ctx.strokeStyle = color;
      ctx.lineWidth = 3;
      ctx.beginPath();
      points.forEach((point, index) => {
        const x = getX(point.timestamp);
        const y = getY(point.value);
        if (index === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      });
      ctx.stroke();

      const targetMarkers = Math.max(4, Math.floor(plotWidth / 260));
      const step = Math.max(1, Math.ceil(points.length / targetMarkers));
      const markerMode = options.markerMode || 'sparse';
      const radius = points.length <= 20 ? 3.6 : points.length <= 60 ? 3.0 : points.length <= 140 ? 2.5 : 2.1;
      points.forEach((point, index) => {
        const isRegularMarker = index % step === 0;
        const isTransitionMarker =
          index === 0 ||
          index === points.length - 1 ||
          (index > 0 && points[index - 1].value !== point.value) ||
          (index < points.length - 1 && points[index + 1].value !== point.value);
        if (markerMode === 'transitions' && !isTransitionMarker) return;
        if (markerMode !== 'transitions' && !isRegularMarker && !isTransitionMarker) return;
        const x = getX(point.timestamp);
        const y = getY(point.value);
        ctx.beginPath();
        ctx.fillStyle = color;
        ctx.strokeStyle = '#11161d';
        ctx.lineWidth = 1.3;
        ctx.arc(x, y, isTransitionMarker ? radius + 0.9 : radius, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
      });

      if (highlight) {
        ctx.save();
        ctx.strokeStyle = 'rgba(231,233,238,.35)';
        ctx.lineWidth = 1;
        ctx.setLineDash([4, 4]);
        ctx.beginPath();
        ctx.moveTo(highlight.x, top);
        ctx.lineTo(highlight.x, height - bottom);
        ctx.stroke();
        ctx.restore();

        ctx.beginPath();
        ctx.fillStyle = color;
        ctx.strokeStyle = '#e7e9ee';
        ctx.lineWidth = 2;
        ctx.arc(highlight.x, highlight.y, 6, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
      }

      if (legend) {
        legend.innerHTML = '<span class="legend-item"><span class="legend-swatch" style="background:' + color + '"></span>' + escapeHtml(label) + '</span>';
      }
    }

    function redraw(highlight) {
      drawBase();
      drawData(highlight);
    }

    redraw();

    if (tooltip) {
      const targetMarkers = Math.max(4, Math.floor(plotWidth / 260));
      const step = Math.max(1, Math.ceil(points.length / targetMarkers));
      const markerMode = options.markerMode || 'sparse';
      const hitPoints = points
        .map((point, index) => ({ x: getX(point.timestamp), y: getY(point.value), point, index }))
        .filter(item =>
          (markerMode !== 'transitions' && item.index % step === 0) ||
          item.index === 0 ||
          item.index === points.length - 1 ||
          (item.index > 0 && points[item.index - 1].value !== item.point.value) ||
          (item.index < points.length - 1 && points[item.index + 1].value !== item.point.value)
        );
      canvas.onmousemove = event => {
        const rect = canvas.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;
        let best = null, bestDistance = Infinity;
        hitPoints.forEach(item => {
          const dx = item.x - x;
          const dy = item.y - y;
          const distance = Math.sqrt(dx * dx + dy * dy);
          if (distance < bestDistance) {
            bestDistance = distance;
            best = item;
          }
        });
        if (!best || bestDistance > 42) {
          tooltip.style.display = 'none';
          redraw();
          return;
        }
        redraw(best);
        tooltip.style.display = 'block';
        tooltip.style.left = Math.min(best.x + 14, width - 220) + 'px';
        tooltip.style.top = Math.max(top + 8, best.y - 54) + 'px';
        const fullTime = new Date(best.point.timestamp).toLocaleString();
        tooltip.innerHTML = '<strong>' + escapeHtml(label) + '</strong><br>Time: ' + escapeHtml(fullTime) + '<br>Status: ' + (best.point.value ? 'Connected' : 'Disconnected');
      };
      canvas.onmouseleave = () => {
        tooltip.style.display = 'none';
        redraw();
      };
    }
  }


  function rememberBinaryStatus(storageKey, value, timestamp, windowMinutes) {
    const key = 'iot-status-history:' + storageKey;
    const now = timestamp ? new Date(timestamp) : new Date();
    const cutoff = now.getTime() - (windowMinutes || 1440) * 60 * 1000;
    let items = [];
    try {
      items = JSON.parse(localStorage.getItem(key) || '[]');
      if (!Array.isArray(items)) items = [];
    } catch (_) {
      items = [];
    }

    items = items.filter(item => {
      const t = new Date(item.timestamp).getTime();
      return Number.isFinite(t) && t >= cutoff;
    });

    const normalizedValue = Boolean(value);
    const stamp = now.toISOString();
    const previous = items.length ? items[items.length - 1] : null;
    const previousTime = previous ? new Date(previous.timestamp).getTime() : 0;
    const shouldAppend = !previous ||
      previous.value !== normalizedValue ||
      !Number.isFinite(previousTime) ||
      Math.abs(now.getTime() - previousTime) >= 60000;

    if (shouldAppend) {
      items.push({ timestamp: stamp, value: normalizedValue });
    } else if (items.length === 1) {
      // Keep the first real observation and add a second point for a short, honest line segment.
      items.push({ timestamp: stamp, value: normalizedValue });
    } else {
      // Keep the current edge fresh without inventing data before the first real observation.
      items[items.length - 1] = { timestamp: stamp, value: normalizedValue };
    }

    try {
      localStorage.setItem(key, JSON.stringify(items.slice(-300)));
    } catch (_) {
      // Local storage is optional. The chart still works with the current in-memory samples.
    }
    return items;
  }


function renderConnectionOverview(options) {
    const current = Boolean(options.connected);
    const statusText = current ? 'connected' : 'disconnected';
    const cards = [
      { label: 'Name', value: options.name || 'Component', hint: options.nameHint || 'Runtime component' },
      { label: 'Host', value: options.host || 'n/a', hint: options.hostHint || 'Endpoint host' },
      { label: 'Port', value: String(options.port || 'n/a'), hint: options.portHint || 'Endpoint port' },
      { label: 'Current status', value: statusText, hint: options.statusHint || 'Latest observed state' }
    ];

    (options.extraCards || []).forEach(item => cards.push(item));
    renderOverviewCards(options.cardsElementId, cards);

    const observedAt = new Date();
    const windowMinutes = options.windowMinutes || 1440;
    const points = rememberBinaryStatus(
      options.historyKey || options.name || 'component',
      current,
      observedAt.toISOString(),
      windowMinutes
    );

    renderBinaryStatusChart({
      canvasId: options.canvasId,
      legendId: options.legendId,
      tooltipId: options.tooltipId,
      height: options.height || 300,
      label: (options.name || 'Component') + ' connection',
      points,
      color: options.color || '#ff8a1d',
      domainStart: new Date(observedAt.getTime() - windowMinutes * 60 * 1000).toISOString(),
      domainEnd: observedAt.toISOString()
    });
  }


  window.IoTUI = {
    escapeHtml,
    statusDot,
    renderStatusDetails,
    renderOverviewCards,
    renderGroupedLogs,
    renderBinaryStatusChart,
    rememberBinaryStatus,
    renderConnectionOverview
  };
})();
