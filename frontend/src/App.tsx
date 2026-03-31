import { Sidebar } from './components/layout/Sidebar'
import { TopBar } from './components/layout/TopBar'
import { SQLPanel } from './components/sql/SQLPanel'
import { NoSQLPanel } from './components/nosql/NoSQLPanel'
import { KVPanel } from './components/kv/KVPanel'
import { useStore } from './hooks/useStore'

function App() {
  const store = useStore()

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar
        connections={store.connections}
        activeConnectionId={store.activeConnectionId}
        onSelect={store.setActiveConnectionId}
        collapsed={store.sidebarCollapsed}
        onToggle={store.toggleSidebar}
        onConnectionsChanged={store.refreshConnections}
      />

      <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {store.activeConnection ? (
          <>
            <TopBar
              connection={store.activeConnection}
              category={store.activeCategory}
              onConnectionsChanged={store.refreshConnections}
            />
            <div className="flex-1 overflow-hidden">
              {store.activeCategory === 'sql' && <SQLPanel connectionId={store.activeConnection.id} />}
              {store.activeCategory === 'nosql' && <NoSQLPanel connectionId={store.activeConnection.id} />}
              {store.activeCategory === 'kv' && <KVPanel connectionId={store.activeConnection.id} />}
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-text-muted text-sm">
            No connections yet. Click "New connection" to get started.
          </div>
        )}
      </main>
    </div>
  )
}

export default App
