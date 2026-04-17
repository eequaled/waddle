<script>
  import { onMount, onDestroy } from 'svelte';
  import { GetCaptureStatus } from '../wailsjs/go/main/App.js';

  let statusText = "Initializing...";
  let statusColor = "var(--text-secondary)";
  let pollInterval;

  async function pollStatus() {
    try {
      const stats = await GetCaptureStatus();
      if (stats && stats.running !== undefined) {
        if (stats.running) {
          statusText = `Waddle v2 • Windows • Tracker Active (${stats.source || 'ETW'})`;
          statusColor = "#4caf50"; // Green for active
        } else {
          statusText = "Waddle v2 • Tracker Paused";
          statusColor = "#ff9800"; // Orange for paused
        }
      } else {
        statusText = "Waddle v2 • Inactive";
        statusColor = "var(--text-secondary)";
      }
    } catch (e) {
      statusText = "Waddle v2 • Disconnected";
      statusColor = "#f44336"; // Red for error
    }
  }

  onMount(() => {
    pollStatus();
    pollInterval = setInterval(pollStatus, 2000);
  });

  onDestroy(() => {
    if (pollInterval) clearInterval(pollInterval);
  });
</script>

<div class="status-bar">
  <div class="status-indicator" style="background-color: {statusColor};"></div>
  <span>{statusText}</span>
</div>

<style>
  .status-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    font-weight: 500;
  }
  
  .status-indicator {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    transition: background-color 0.3s ease;
  }
</style>
