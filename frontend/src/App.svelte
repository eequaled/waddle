<script>
  import Sidebar from './Sidebar.svelte';
  import StatusBar from './StatusBar.svelte';
  import AgentView from './AgentView.svelte';
  import MemoryView from './MemoryView.svelte';
  import SearchView from './SearchView.svelte';
  import ChatView from './ChatView.svelte';

  let currentView = 'agent';

  function setView(view) {
    currentView = view;
  }
</script>

<div class="app-container">
  <Sidebar {currentView} onSelect={setView} />
  
  <main class="main-content">
    {#if currentView === 'agent'}
      <AgentView />
    {:else if currentView === 'memory'}
      <MemoryView />
    {:else if currentView === 'search'}
      <SearchView />
    {:else if currentView === 'chat'}
      <ChatView />
    {/if}
  </main>

  <StatusBar />
</div>
