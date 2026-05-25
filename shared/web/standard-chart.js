
(function () {
  const palette = ['#ff8a1d', '#28c76f', '#febe3b', '#57a6ff', '#ff4d4f', '#c084fc', '#14b8a6', '#f472b6', '#84cc16', '#fb923c'];

  function setupCanvas(canvas, height) {
    const ratio = window.devicePixelRatio || 1;
    const width = canvas.clientWidth || canvas.width || 900;
    const targetHeight = height || canvas.clientHeight || 340;
    canvas.width = Math.floor(width * ratio);
    canvas.height = Math.floor(targetHeight * ratio);
    const ctx = canvas.getContext('2d');
    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    return { ctx, width, height: targetHeight };
  }

  function formatTimeLabel(stamp) {
    if (!stamp) return '';
    const date = new Date(stamp);
    if (Number.isNaN(date.getTime())) return String(stamp).substring(11, 19);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }

  function createChart(options) {
    const canvas = document.getElementById(options.canvasId);
    const legend = options.legendId ? document.getElementById(options.legendId) : null;
    const tooltip = options.tooltipId ? document.getElementById(options.tooltipId) : null;
    const state = { series: [], hidden: new Set(), hitPoints: [], yLabel: options.yLabel || 'value' };

    function visibleSeries() {
      return state.series.filter(item => !state.hidden.has(item.key));
    }

    function draw(series, yLabel) {
      state.series = (series || []).map((item, index) => ({
        key: item.key || item.label || String(index),
        label: item.label || item.key || String(index),
        color: item.color || palette[index % palette.length],
        points: item.points || []
      }));
      state.yLabel = yLabel || state.yLabel || 'value';
      render();
    }

    function render(highlight) {
      const { ctx, width, height } = setupCanvas(canvas, options.height || 340);
      const active = visibleSeries();
      state.hitPoints = [];
      ctx.clearRect(0, 0, width, height);

      if (!active.length) {
        ctx.fillStyle = '#8a9099';
        ctx.font = '14px Inter, system-ui, sans-serif';
        ctx.fillText('No data available or all series are hidden.', 20, 30);
        renderLegend();
        return;
      }

      const values = active.flatMap(item => item.points.map(point => Number(point.value))).filter(Number.isFinite);
      let minValue = Math.min(...values);
      let maxValue = Math.max(...values);
      if (minValue === maxValue) {
        minValue -= 1;
        maxValue += 1;
      }

      const left = 68;
      const right = 28;
      const top = 22;
      const bottom = 58;
      const plotWidth = width - left - right;
      const plotHeight = height - top - bottom;

      ctx.strokeStyle = '#2a2f38';
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(left, top);
      ctx.lineTo(left, height - bottom);
      ctx.lineTo(width - right, height - bottom);
      ctx.stroke();

      ctx.font = '12px Inter, system-ui, sans-serif';
      for (let i = 0; i <= 4; i++) {
        const y = height - bottom - (plotHeight / 4) * i;
        const value = minValue + ((maxValue - minValue) / 4) * i;
        ctx.strokeStyle = 'rgba(42,47,56,.75)';
        ctx.beginPath();
        ctx.moveTo(left, y);
        ctx.lineTo(width - right, y);
        ctx.stroke();
        ctx.fillStyle = '#8a9099';
        ctx.fillText(value.toFixed(2), 8, y + 4);
      }

      const allPoints = active.flatMap(item => item.points);
      const timestamps = allPoints.map(point => new Date(point.timestamp).getTime()).filter(Number.isFinite);
      const minTime = timestamps.length ? Math.min(...timestamps) : 0;
      const maxTime = timestamps.length ? Math.max(...timestamps) : 0;
      const useTime = maxTime > minTime;

      if (useTime) {
        [minTime, (minTime + maxTime) / 2, maxTime].forEach((time, index) => {
          const x = left + (plotWidth / 2) * index;
          ctx.strokeStyle = 'rgba(42,47,56,.5)';
          ctx.beginPath();
          ctx.moveTo(x, top);
          ctx.lineTo(x, height - bottom);
          ctx.stroke();
          ctx.fillStyle = '#8a9099';
          ctx.fillText(formatTimeLabel(new Date(time).toISOString()), x - 28, height - 26);
        });
      }

      function markerStep(pointCount) {
        const targetMarkers = Math.max(8, Math.floor(plotWidth / 120));
        return Math.max(1, Math.ceil(pointCount / targetMarkers));
      }

      function markerRadius(pointCount) {
        if (pointCount <= 20) return 3.6;
        if (pointCount <= 60) return 3.0;
        if (pointCount <= 140) return 2.5;
        return 2.1;
      }

      function isShapePoint(points, index) {
        if (index === 0 || index === points.length - 1) return true;
        const prev = Number(points[index - 1].point.value);
        const current = Number(points[index].point.value);
        const next = Number(points[index + 1].point.value);
        if (!Number.isFinite(prev) || !Number.isFinite(current) || !Number.isFinite(next)) return false;
        const before = current - prev;
        const after = next - current;
        const range = Math.max(1e-9, maxValue - minValue);
        const jumpThreshold = range * 0.08;
        const isTurn = (before > 0 && after < 0) || (before < 0 && after > 0);
        const isJump = Math.abs(before) >= jumpThreshold || Math.abs(after) >= jumpThreshold;
        return isTurn || isJump;
      }

      active.forEach(item => {
        const visiblePoints = [];
        ctx.strokeStyle = item.color;
        ctx.lineWidth = 2;
        ctx.beginPath();

        item.points.forEach((point, index) => {
          const time = new Date(point.timestamp).getTime();
          const x = useTime ? left + ((time - minTime) / Math.max(1, maxTime - minTime)) * plotWidth : left + (index / Math.max(1, item.points.length - 1)) * plotWidth;
          const y = height - bottom - ((Number(point.value) - minValue) / (maxValue - minValue)) * plotHeight;
          if (index === 0) ctx.moveTo(x, y);
          else ctx.lineTo(x, y);
          visiblePoints.push({ x, y, series: item, point, color: item.color, index });
        });
        ctx.stroke();

        const step = markerStep(visiblePoints.length);
        const radius = markerRadius(visiblePoints.length);
        visiblePoints.forEach((item, index) => {
          const isRegularMarker = index % step === 0;
          const isImportantMarker = isShapePoint(visiblePoints, index);
          if (!isRegularMarker && !isImportantMarker) return;
          state.hitPoints.push(item);
          ctx.beginPath();
          ctx.fillStyle = item.color;
          ctx.strokeStyle = '#11161d';
          ctx.lineWidth = 1.3;
          ctx.arc(item.x, item.y, isImportantMarker ? radius + 0.8 : radius, 0, Math.PI * 2);
          ctx.fill();
          ctx.stroke();
        });
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
        ctx.fillStyle = highlight.color;
        ctx.strokeStyle = '#e7e9ee';
        ctx.lineWidth = 2;
        ctx.arc(highlight.x, highlight.y, 6, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
      }

      ctx.fillStyle = '#8a9099';
      ctx.fillText('time', width - 54, height - 10);
      ctx.save();
      ctx.translate(16, height / 2);
      ctx.rotate(-Math.PI / 2);
      ctx.fillText(state.yLabel, 0, 0);
      ctx.restore();

      renderLegend();
    }

    function renderLegend() {
      if (!legend) return;
      legend.innerHTML = '';
      state.series.forEach(item => {
        const el = document.createElement('span');
        el.className = 'legend-item ' + (state.hidden.has(item.key) ? 'off' : '');
        el.innerHTML = '<span class="legend-swatch" style="background:' + item.color + '"></span>' + escapeHtml(item.label);
        el.onclick = () => {
          if (state.hidden.has(item.key)) state.hidden.delete(item.key);
          else state.hidden.add(item.key);
          render();
        };
        legend.appendChild(el);
      });
    }

    function attachTooltip() {
      if (!tooltip) return;
      canvas.addEventListener('mousemove', event => {
        const rect = canvas.getBoundingClientRect();
        const mouseX = event.clientX - rect.left;
        const mouseY = event.clientY - rect.top;
        let best = null;
        let bestScore = Infinity;

        state.hitPoints.forEach(item => {
          const dx = item.x - mouseX;
          const dy = item.y - mouseY;
          const score = Math.sqrt(dx * dx + dy * dy);
          if (score < bestScore) {
            bestScore = score;
            best = item;
          }
        });

        if (!best || bestScore > 44) {
          tooltip.style.display = 'none';
          render();
          return;
        }

        render(best);

        const width = canvas.clientWidth || canvas.width || 900;
        const top = 18;

        tooltip.style.left = Math.min(best.x + 14, width - 230) + 'px';
        tooltip.style.top = Math.max(top + 8, best.y - 58) + 'px';
        tooltip.style.display = 'block';
        const fullTime = best.point.timestamp ? new Date(best.point.timestamp).toLocaleString() : '';
        tooltip.innerHTML = '<strong>' + escapeHtml(best.series.label) + '</strong><br>Time: ' + escapeHtml(fullTime || formatTimeLabel(best.point.timestamp)) + '<br>Value: ' + Number(best.point.value).toFixed(3);
      });
      canvas.addEventListener('mouseleave', () => {
        tooltip.style.display = 'none';
        render();
      });
    }

    attachTooltip();
    return { draw, render, hidden: state.hidden };
  }

  function escapeHtml(value) {
    return String(value)
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#039;');
  }

  function fillSelect(select, options) {
    const current = select.value;
    select.innerHTML = (options || []).map(option => '<option value="' + escapeHtml(option.value) + '">' + escapeHtml(option.label) + '</option>').join('');
    if ([...select.options].some(option => option.value === current)) select.value = current;
  }

  window.IoTStandardChart = { createChart, palette, escapeHtml, formatTimeLabel, fillSelect };
})();
