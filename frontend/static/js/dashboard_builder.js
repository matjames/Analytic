async function initDashboardBuilder() {
  const container = document.getElementById('plotlyDashboardContainer');
  const dsSelect = document.getElementById('builderDatasetSelect');
  if (!container || !dsSelect) return;

  try {
    const res = await fetch('/api/kaggle/datasets');
    const data = await res.json();
    dsSelect.innerHTML = '<option value="">-- Select Dataset to Build Visualization --</option>';
    (data.tables || []).forEach(t => {
      const opt = document.createElement('option');
      opt.value = t;
      opt.innerText = `Dataset: ${t}`;
      dsSelect.appendChild(opt);
    });
  } catch (e) {
    console.error("Failed to populate builder datasets:", e);
  }
}

async function onBuilderDatasetChange() {
  const dsSelect = document.getElementById('builderDatasetSelect');
  const colXSelect = document.getElementById('builderColXSelect');
  const colYSelect = document.getElementById('builderColYSelect');
  if (!dsSelect || !dsSelect.value) return;

  const table = dsSelect.value;
  try {
    const res = await fetch(`/api/kaggle/schema/${table}`);
    const summary = await res.json();

    colXSelect.innerHTML = '';
    colYSelect.innerHTML = '';

    (summary.profiling || []).forEach(col => {
      const optX = document.createElement('option');
      optX.value = col.column_name;
      optX.innerText = `X: ${col.column_name} (${col.data_type})`;
      colXSelect.appendChild(optX);

      const optY = document.createElement('option');
      optY.value = col.column_name;
      optY.innerText = `Y: ${col.column_name} (${col.data_type})`;
      colYSelect.appendChild(optY);
    });

    renderPlotlyChart();
  } catch (e) {
    console.error("Schema fetch error:", e);
  }
}

async function renderPlotlyChart() {
  const table = document.getElementById('builderDatasetSelect').value;
  const colX = document.getElementById('builderColXSelect').value;
  const colY = document.getElementById('builderColYSelect').value;
  const chartType = document.getElementById('builderChartTypeSelect').value;
  const container = document.getElementById('plotlyDashboardContainer');

  if (!table || !colX || !colY) return;

  try {
    const res = await fetch(`/api/kaggle/schema/${table}`);
    const data = await res.json();
    const records = data.sample_records || [];

    const xVals = records.map(r => r[colX]);
    const yVals = records.map(r => r[colY]);

    const trace = {
      x: xVals,
      y: yVals,
      type: chartType === 'bar' ? 'bar' : chartType === 'scatter' ? 'scatter' : 'histogram',
      mode: chartType === 'scatter' ? 'lines+markers' : undefined,
      marker: { color: '#00f2fe' }
    };

    const layout = {
      title: { text: `Dynamic Metadata Chart: ${table} (${colX} vs ${colY})`, font: { color: '#f0f4f8' } },
      paper_bgcolor: 'rgba(10, 14, 23, 0.75)',
      plot_bgcolor: 'rgba(18, 24, 36, 0.5)',
      xaxis: { title: colX, color: '#8a99ad', gridcolor: 'rgba(255,255,255,0.08)' },
      yaxis: { title: colY, color: '#8a99ad', gridcolor: 'rgba(255,255,255,0.08)' },
      margin: { t: 50, b: 40, l: 50, r: 20 }
    };

    Plotly.newPlot(container, [trace], layout, { responsive: true });
  } catch (e) {
    console.error("Plotly rendering error:", e);
  }
}

async function saveDashboardLayout() {
  const table = document.getElementById('builderDatasetSelect').value;
  const colX = document.getElementById('builderColXSelect').value;
  const colY = document.getElementById('builderColYSelect').value;
  const chartType = document.getElementById('builderChartTypeSelect').value;

  const metadata = {
    table: table,
    colX: colX,
    colY: colY,
    chartType: chartType,
    updated_at: new Date().toISOString()
  };

  await fetch('/api/dashboard/save', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ dashboard_id: 'default_layout', metadata: metadata })
  });

  alert("Dashboard metadata layout saved to data/dashboards/default_layout.json!");
}

document.addEventListener('DOMContentLoaded', () => {
  initDashboardBuilder();
});
