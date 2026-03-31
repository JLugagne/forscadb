// api.ts - Wails backend API wrapper
// At runtime, Wails exposes: window.go.main.App.<method>(args)

// Declare the Wails runtime interface
declare global {
  interface Window {
    go: {
      main: {
        App: {
          // Connection management
          ListConnections(): Promise<any[]>
          CreateConnection(input: any): Promise<any>
          UpdateConnection(input: any): Promise<any>
          DeleteConnection(id: string): Promise<void>
          ConnectToDatabase(id: string): Promise<void>
          DisconnectFromDatabase(id: string): Promise<void>
          GetConnection(id: string): Promise<any>
          TestConnection(input: any): Promise<void>
          // SQL
          GetTables(connID: string): Promise<any[]>
          GetViews(connID: string): Promise<any[]>
          GetFunctions(connID: string): Promise<any[]>
          GetTriggers(connID: string): Promise<any[]>
          GetSequences(connID: string): Promise<any[]>
          GetEnums(connID: string): Promise<any[]>
          ExecuteQuery(connID: string, query: string): Promise<any>
          GetTableData(connID: string, schema: string, table: string, limit: number, offset: number): Promise<any>
          GetQueryHistory(connID: string): Promise<any[]>
          DropTable(connID: string, schema: string, table: string): Promise<void>
          AddColumn(connID: string, schema: string, table: string, name: string, colType: string, nullable: boolean, defaultVal: string): Promise<void>
          RefreshMaterializedView(connID: string, schema: string, name: string): Promise<void>
          RenameColumn(connID: string, schema: string, table: string, oldName: string, newName: string): Promise<void>
          AlterColumnType(connID: string, schema: string, table: string, column: string, newType: string): Promise<void>
          DropColumn(connID: string, schema: string, table: string, column: string): Promise<void>
          SetColumnNullable(connID: string, schema: string, table: string, column: string, nullable: boolean): Promise<void>
          SetColumnDefault(connID: string, schema: string, table: string, column: string, defaultVal: string): Promise<void>
          ExplainQuery(connID: string, query: string, analyze: boolean): Promise<any>
          // NoSQL
          GetCollections(connID: string): Promise<any[]>
          GetDocuments(connID: string, collection: string, filter: string, limit: number): Promise<any[]>
          InsertDocument(connID: string, collection: string, doc: any): Promise<any>
          UpdateDocument(connID: string, collection: string, id: string, doc: any): Promise<any>
          DeleteDocument(connID: string, collection: string, id: string): Promise<void>
          CreateCollection(connID: string, name: string): Promise<void>
          DropCollection(connID: string, name: string): Promise<void>
          // KV
          GetKVStats(connID: string): Promise<any>
          GetKeys(connID: string, pattern: string, limit: number): Promise<any[]>
          GetKVEntry(connID: string, key: string): Promise<any>
          SetKVEntry(connID: string, key: string, value: string, ttl: number | null): Promise<void>
          DeleteKVEntry(connID: string, key: string): Promise<void>
          // Sidebar tree
          GetSidebarTree(): Promise<string>
          SaveSidebarTree(tree: string): Promise<void>
        }
      }
    }
  }
}

const app = () => window.go.main.App

// Connection management
export const listConnections = () => app().ListConnections()
export const createConnection = (input: any) => app().CreateConnection(input)
export const updateConnection = (input: any) => app().UpdateConnection(input)
export const deleteConnection = (id: string) => app().DeleteConnection(id)
export const connectToDatabase = (id: string) => app().ConnectToDatabase(id)
export const disconnectFromDatabase = (id: string) => app().DisconnectFromDatabase(id)
export const testConnection = (input: any) => app().TestConnection(input)

// SQL
export const getTables = (connID: string) => app().GetTables(connID)
export const getViews = (connID: string) => app().GetViews(connID)
export const getFunctions = (connID: string) => app().GetFunctions(connID)
export const getTriggers = (connID: string) => app().GetTriggers(connID)
export const getSequences = (connID: string) => app().GetSequences(connID)
export const getEnums = (connID: string) => app().GetEnums(connID)
export const executeQuery = (connID: string, query: string) => app().ExecuteQuery(connID, query)
export const getTableData = (connID: string, schema: string, table: string, limit: number, offset: number) => app().GetTableData(connID, schema, table, limit, offset)
export const getQueryHistory = (connID: string) => app().GetQueryHistory(connID)
export const dropTable = (connID: string, schema: string, table: string) => app().DropTable(connID, schema, table)
export const addColumn = (connID: string, schema: string, table: string, name: string, colType: string, nullable: boolean, defaultVal: string) => app().AddColumn(connID, schema, table, name, colType, nullable, defaultVal)
export const refreshMaterializedView = (connID: string, schema: string, name: string) => app().RefreshMaterializedView(connID, schema, name)
export const renameColumn = (connID: string, schema: string, table: string, oldName: string, newName: string) => app().RenameColumn(connID, schema, table, oldName, newName)
export const alterColumnType = (connID: string, schema: string, table: string, column: string, newType: string) => app().AlterColumnType(connID, schema, table, column, newType)
export const dropColumn = (connID: string, schema: string, table: string, column: string) => app().DropColumn(connID, schema, table, column)
export const setColumnNullable = (connID: string, schema: string, table: string, column: string, nullable: boolean) => app().SetColumnNullable(connID, schema, table, column, nullable)
export const setColumnDefault = (connID: string, schema: string, table: string, column: string, defaultVal: string) => app().SetColumnDefault(connID, schema, table, column, defaultVal)
export const explainQuery = (connID: string, query: string, analyze: boolean) => app().ExplainQuery(connID, query, analyze)

// NoSQL
export const getCollections = (connID: string) => app().GetCollections(connID)
export const getDocuments = (connID: string, collection: string, filter: string, limit: number) => app().GetDocuments(connID, collection, filter, limit)
export const insertDocument = (connID: string, collection: string, doc: any) => app().InsertDocument(connID, collection, doc)
export const updateDocument = (connID: string, collection: string, id: string, doc: any) => app().UpdateDocument(connID, collection, id, doc)
export const deleteDocument = (connID: string, collection: string, id: string) => app().DeleteDocument(connID, collection, id)
export const createCollection = (connID: string, name: string) => app().CreateCollection(connID, name)
export const dropCollection = (connID: string, name: string) => app().DropCollection(connID, name)

// KV
export const getKVStats = (connID: string) => app().GetKVStats(connID)
export const getKeys = (connID: string, pattern: string, limit: number) => app().GetKeys(connID, pattern, limit)
export const getKVEntry = (connID: string, key: string) => app().GetKVEntry(connID, key)
export const setKVEntry = (connID: string, key: string, value: string, ttl: number | null) => app().SetKVEntry(connID, key, value, ttl)
export const deleteKVEntry = (connID: string, key: string) => app().DeleteKVEntry(connID, key)

// Sidebar tree
export const getSidebarTree = () => app().GetSidebarTree()
export const saveSidebarTree = (tree: string) => app().SaveSidebarTree(tree)
