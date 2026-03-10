<script>
  import { onMount } from 'svelte';
  import { GetSessions, GetAppDetails } from '../wailsjs/go/main/App.js';

  let sessions = [];
  let selectedDate = null;
  let appDetails = [];
  let loading = true;

  onMount(async () => {
    try {
      const res = await GetSessions();
      sessions = res || [];
      if (sessions.length > 0) {
        await selectSession(sessions[0].date);
      }
    } catch (e) {
      console.error('Failed to load sessions:', e);
    } finally {
      loading = false;
    }
  });

  async function selectSession(date) {
    selectedDate = date;
    try {
      const details = await GetAppDetails(date);
      appDetails = details || [];
    } catch (e) {
      console.error(`Failed to load details for ${date}:`, e);
      appDetails = [];
    }
  }
</script>

<div class="memory-container">
  <div class="sidebar-list">
    <h2 class="view-title">Sessions</h2>
    {#if loading}
      <div class="loading">Loading memory...</div>
    {:else if sessions.length === 0}
      <div class="empty-state">No sessions recorded yet.</div>
    {:else}
      <ul class="session-list">
        {#each sessions as session}
          <li>
            <button 
              class="session-btn {selectedDate === session.date ? 'active' : ''}" 
              on:click={() => selectSession(session.date)}
            >
              <div class="session-date">{session.date}</div>
              <div class="session-title">{session.customTitle || "Untitled Session"}</div>
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>

  <div class="detail-pane">
    {#if selectedDate}
      <h2 class="view-title">Details for {selectedDate}</h2>
      
      {#if appDetails.length === 0}
        <div class="empty-state">No activity details found.</div>
      {:else}
        <div class="cards-grid">
          {#each appDetails as detail}
            <div class="app-card">
              <h3 class="app-name">{detail.appName}</h3>
              <p class="app-stats">{detail.blockCount} blocks recorded</p>
            </div>
          {/each}
        </div>
      {/if}
    {:else}
      <div class="empty-state">Select a session to view details.</div>
    {/if}
  </div>
</div>

<style>
  .memory-container {
    display: flex;
    gap: 24px;
    height: 100%;
  }

  .sidebar-list {
    flex: 0 0 250px;
    border-right: 1px solid var(--border);
    padding-right: 16px;
    display: flex;
    flex-direction: column;
  }

  .detail-pane {
    flex: 1;
    display: flex;
    flex-direction: column;
  }

  .view-title {
    font-size: 14px;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--text-secondary);
    margin-bottom: 16px;
  }

  .session-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
    overflow-y: auto;
  }

  .session-btn {
    width: 100%;
    text-align: left;
    background: transparent;
    border: 1px solid transparent;
    border-radius: 6px;
    padding: 12px;
    color: var(--text-primary);
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .session-btn:hover {
    background: var(--surface);
  }

  .session-btn.active {
    background: var(--surface);
    border-color: var(--cyan);
  }

  .session-date {
    font-weight: 600;
    font-size: 14px;
    margin-bottom: 4px;
    color: var(--text-primary);
  }

  .session-title {
    font-size: 12px;
    color: var(--text-secondary);
  }

  .cards-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 16px;
  }

  .app-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 16px;
    transition: transform 0.2s ease, border-color 0.2s ease;
  }

  .app-card:hover {
    transform: translateY(-2px);
    border-color: var(--cyan);
  }

  .app-name {
    font-size: 16px;
    font-weight: 600;
    margin: 0 0 8px 0;
  }

  .app-stats {
    font-size: 12px;
    color: var(--text-secondary);
    margin: 0;
  }

  .loading, .empty-state {
    padding: 24px;
    text-align: center;
    color: var(--text-secondary);
    font-size: 14px;
    background: var(--surface);
    border-radius: 8px;
    border: 1px dashed var(--border);
  }
</style>
