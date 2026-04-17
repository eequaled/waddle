<script>
  export let currentView;
  export let onSelect;

  const items = [
    { id: 'agent', label: 'Agent', icon: '🤖' },
    { id: 'memory', label: 'Memory', icon: '🧠' },
    { id: 'search', label: 'Search', icon: '🔍' },
    { id: 'chat', label: 'Chat', icon: '💬' }
  ];
</script>

<aside class="sidebar">
  <div style="padding: 24px; font-weight: bold; font-size: 20px; color: var(--accent);">Waddle v2</div>
  
  <nav style="flex-grow: 1;">
    {#each items as item}
      <button
        type="button"
        class="nav-item {currentView === item.id ? 'active' : ''}"
        on:click={() => onSelect(item.id)}
      >
        <span>{item.icon}</span>
        <span>{item.label}</span>
      </button>
    {/each}
  </nav>

  <div style="padding: 24px; color: var(--text-secondary); font-size: 12px;">
    System v0.1.0
  </div>
</aside>
