export namespace main {
	
	export class AppDetail {
	    appName: string;
	    blockCount: number;
	
	    static createFrom(source: any = {}) {
	        return new AppDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appName = source["appName"];
	        this.blockCount = source["blockCount"];
	    }
	}
	export class HealthStatusResponse {
	    status: string;
	    timestamp?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthStatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.timestamp = source["timestamp"];
	        this.error = source["error"];
	    }
	}

}

export namespace pipeline {
	
	export class PipelineStats {
	    running: boolean;
	    source: string;
	    etwFallbackMode: boolean;
	    droppedEvents: number;
	    activityBufferSize: number;
	    ocrBufferSize: number;
	
	    static createFrom(source: any = {}) {
	        return new PipelineStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.source = source["source"];
	        this.etwFallbackMode = source["etwFallbackMode"];
	        this.droppedEvents = source["droppedEvents"];
	        this.activityBufferSize = source["activityBufferSize"];
	        this.ocrBufferSize = source["ocrBufferSize"];
	    }
	}

}

export namespace storage {
	
	export class ActivityBlock {
	    id: number;
	    appActivityId: number;
	    blockId: string;
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime: any;
	    ocrText: string;
	    microSummary: string;
	    captureSource: string;
	    structuredMetadata: string;
	
	    static createFrom(source: any = {}) {
	        return new ActivityBlock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.appActivityId = source["appActivityId"];
	        this.blockId = source["blockId"];
	        this.startTime = this.convertValues(source["startTime"], null);
	        this.endTime = this.convertValues(source["endTime"], null);
	        this.ocrText = source["ocrText"];
	        this.microSummary = source["microSummary"];
	        this.captureSource = source["captureSource"];
	        this.structuredMetadata = source["structuredMetadata"];
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
	export class AppActivity {
	    id: number;
	    sessionId: number;
	    appName: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    blocks?: ActivityBlock[];
	
	    static createFrom(source: any = {}) {
	        return new AppActivity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.appName = source["appName"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.blocks = this.convertValues(source["blocks"], ActivityBlock);
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
	export class ManualNote {
	    id: number;
	    sessionId: number;
	    content: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ManualNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.content = source["content"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class Session {
	    id: number;
	    date: string;
	    customTitle: string;
	    customSummary: string;
	    originalSummary: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    entitiesJson: string;
	    synthesisStatus: string;
	    aiSummary: string;
	    aiBullets: string;
	    activities?: AppActivity[];
	    manualNotes?: ManualNote[];
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.date = source["date"];
	        this.customTitle = source["customTitle"];
	        this.customSummary = source["customSummary"];
	        this.originalSummary = source["originalSummary"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.entitiesJson = source["entitiesJson"];
	        this.synthesisStatus = source["synthesisStatus"];
	        this.aiSummary = source["aiSummary"];
	        this.aiBullets = source["aiBullets"];
	        this.activities = this.convertValues(source["activities"], AppActivity);
	        this.manualNotes = this.convertValues(source["manualNotes"], ManualNote);
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

