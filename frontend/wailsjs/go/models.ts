export namespace converters {
	
	export class PublicConnection {
	    id: string;
	    name: string;
	    engine: string;
	    category: string;
	    host: string;
	    port: number;
	    user: string;
	    password?: string;
	    database: string;
	    sslMode?: string;
	    status: string;
	    color: string;
	    lastAccess: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicConnection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.engine = source["engine"];
	        this.category = source["category"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.database = source["database"];
	        this.sslMode = source["sslMode"];
	        this.status = source["status"];
	        this.color = source["color"];
	        this.lastAccess = source["lastAccess"];
	    }
	}
	export class PublicExplainRow {
	    text: string;
	    level: number;
	    isNode: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PublicExplainRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.level = source["level"];
	        this.isNode = source["isNode"];
	    }
	}
	export class PublicExplainResult {
	    plan: string;
	    format: string;
	    queryText: string;
	    planRows: PublicExplainRow[];
	
	    static createFrom(source: any = {}) {
	        return new PublicExplainResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plan = source["plan"];
	        this.format = source["format"];
	        this.queryText = source["queryText"];
	        this.planRows = this.convertValues(source["planRows"], PublicExplainRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PublicForeignKey {
	    table: string;
	    column: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicForeignKey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = source["table"];
	        this.column = source["column"];
	    }
	}
	export class PublicFunctionArg {
	    name: string;
	    type: string;
	    mode: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicFunctionArg(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.mode = source["mode"];
	    }
	}
	export class PublicHistoryEntry {
	    id: string;
	    query: string;
	    executedAt: string;
	    duration: number;
	    rowCount: number;
	    status: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicHistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.query = source["query"];
	        this.executedAt = source["executedAt"];
	        this.duration = source["duration"];
	        this.rowCount = source["rowCount"];
	        this.status = source["status"];
	        this.error = source["error"];
	    }
	}
	export class PublicKVEntry {
	    key: string;
	    value: string;
	    type: string;
	    ttl?: number;
	    size: string;
	    encoding: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicKVEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.type = source["type"];
	        this.ttl = source["ttl"];
	        this.size = source["size"];
	        this.encoding = source["encoding"];
	    }
	}
	export class PublicKVStats {
	    totalKeys: number;
	    memoryUsed: string;
	    memoryPeak: string;
	    connectedClients: number;
	    opsPerSec: number;
	    hitRate: number;
	    uptimeDays: number;
	    keyspaceHits: number;
	    keyspaceMisses: number;
	
	    static createFrom(source: any = {}) {
	        return new PublicKVStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalKeys = source["totalKeys"];
	        this.memoryUsed = source["memoryUsed"];
	        this.memoryPeak = source["memoryPeak"];
	        this.connectedClients = source["connectedClients"];
	        this.opsPerSec = source["opsPerSec"];
	        this.hitRate = source["hitRate"];
	        this.uptimeDays = source["uptimeDays"];
	        this.keyspaceHits = source["keyspaceHits"];
	        this.keyspaceMisses = source["keyspaceMisses"];
	    }
	}
	export class PublicNoSQLIndex {
	    name: string;
	    keys: Record<string, number>;
	    unique: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PublicNoSQLIndex(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.keys = source["keys"];
	        this.unique = source["unique"];
	    }
	}
	export class PublicNoSQLCollection {
	    name: string;
	    documentCount: number;
	    avgDocSize: string;
	    totalSize: string;
	    indexes: PublicNoSQLIndex[];
	
	    static createFrom(source: any = {}) {
	        return new PublicNoSQLCollection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.documentCount = source["documentCount"];
	        this.avgDocSize = source["avgDocSize"];
	        this.totalSize = source["totalSize"];
	        this.indexes = this.convertValues(source["indexes"], PublicNoSQLIndex);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PublicQueryResult {
	    columns: string[];
	    rows: any[];
	    rowCount: number;
	    executionTime: number;
	    affectedRows?: number;
	
	    static createFrom(source: any = {}) {
	        return new PublicQueryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	        this.rowCount = source["rowCount"];
	        this.executionTime = source["executionTime"];
	        this.affectedRows = source["affectedRows"];
	    }
	}
	export class PublicSQLColumn {
	    name: string;
	    type: string;
	    nullable: boolean;
	    primaryKey: boolean;
	    defaultValue?: string;
	    foreignKey?: PublicForeignKey;
	
	    static createFrom(source: any = {}) {
	        return new PublicSQLColumn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.nullable = source["nullable"];
	        this.primaryKey = source["primaryKey"];
	        this.defaultValue = source["defaultValue"];
	        this.foreignKey = this.convertValues(source["foreignKey"], PublicForeignKey);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PublicSQLEnum {
	    name: string;
	    schema: string;
	    values: string[];
	
	    static createFrom(source: any = {}) {
	        return new PublicSQLEnum(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.schema = source["schema"];
	        this.values = source["values"];
	    }
	}
	export class PublicSQLFunction {
	    name: string;
	    schema: string;
	    language: string;
	    returnType: string;
	    args: PublicFunctionArg[];
	    volatility: string;
	    definition: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicSQLFunction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.schema = source["schema"];
	        this.language = source["language"];
	        this.returnType = source["returnType"];
	        this.args = this.convertValues(source["args"], PublicFunctionArg);
	        this.volatility = source["volatility"];
	        this.definition = source["definition"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PublicSQLIndex {
	    name: string;
	    columns: string[];
	    unique: boolean;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicSQLIndex(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.columns = source["columns"];
	        this.unique = source["unique"];
	        this.type = source["type"];
	    }
	}
	export class PublicSQLSequence {
	    name: string;
	    schema: string;
	    dataType: string;
	    startValue: number;
	    increment: number;
	    minValue: number;
	    maxValue: number;
	    currentValue: number;
	    cacheSize: number;
	    cycle: boolean;
	    ownedBy?: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicSQLSequence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.schema = source["schema"];
	        this.dataType = source["dataType"];
	        this.startValue = source["startValue"];
	        this.increment = source["increment"];
	        this.minValue = source["minValue"];
	        this.maxValue = source["maxValue"];
	        this.currentValue = source["currentValue"];
	        this.cacheSize = source["cacheSize"];
	        this.cycle = source["cycle"];
	        this.ownedBy = source["ownedBy"];
	    }
	}
	export class PublicSQLTable {
	    name: string;
	    schema: string;
	    columns: PublicSQLColumn[];
	    rowCount: number;
	    size: string;
	    indexes: PublicSQLIndex[];
	
	    static createFrom(source: any = {}) {
	        return new PublicSQLTable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.schema = source["schema"];
	        this.columns = this.convertValues(source["columns"], PublicSQLColumn);
	        this.rowCount = source["rowCount"];
	        this.size = source["size"];
	        this.indexes = this.convertValues(source["indexes"], PublicSQLIndex);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PublicSQLTrigger {
	    name: string;
	    schema: string;
	    table: string;
	    event: string;
	    timing: string;
	    forEach: string;
	    function: string;
	    enabled: boolean;
	    definition: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicSQLTrigger(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.schema = source["schema"];
	        this.table = source["table"];
	        this.event = source["event"];
	        this.timing = source["timing"];
	        this.forEach = source["forEach"];
	        this.function = source["function"];
	        this.enabled = source["enabled"];
	        this.definition = source["definition"];
	    }
	}
	export class PublicSQLView {
	    name: string;
	    schema: string;
	    definition: string;
	    columns: PublicSQLColumn[];
	    materialized: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PublicSQLView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.schema = source["schema"];
	        this.definition = source["definition"];
	        this.columns = this.convertValues(source["columns"], PublicSQLColumn);
	        this.materialized = source["materialized"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

