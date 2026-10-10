export namespace config {
	
	export class FPSSettings {
	    process: string;
	
	    static createFrom(source: any = {}) {
	        return new FPSSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.process = source["process"];
	    }
	}
	export class Hotkeys {
	    timerStart: string;
	    timerSync: string;
	    timerReset: string;
	    overlay: string;
	
	    static createFrom(source: any = {}) {
	        return new Hotkeys(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timerStart = source["timerStart"];
	        this.timerSync = source["timerSync"];
	        this.timerReset = source["timerReset"];
	        this.overlay = source["overlay"];
	    }
	}
	export class Thresholds {
	    pingMs: number;
	    jitterMs: number;
	    lossPct: number;
	    minFps: number;
	
	    static createFrom(source: any = {}) {
	        return new Thresholds(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pingMs = source["pingMs"];
	        this.jitterMs = source["jitterMs"];
	        this.lossPct = source["lossPct"];
	        this.minFps = source["minFps"];
	    }
	}
	export class NetworkSettings {
	    pingTargets: string[];
	    thresholds: Thresholds;
	
	    static createFrom(source: any = {}) {
	        return new NetworkSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pingTargets = source["pingTargets"];
	        this.thresholds = this.convertValues(source["thresholds"], Thresholds);
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
	export class OverlaySettings {
	    x: number;
	    y: number;
	    opacity: number;
	
	    static createFrom(source: any = {}) {
	        return new OverlaySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.opacity = source["opacity"];
	    }
	}
	export class Sharing {
	    asked: boolean;
	    bosses: boolean;
	    relics: boolean;
	    clearTimes: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Sharing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.asked = source["asked"];
	        this.bosses = source["bosses"];
	        this.relics = source["relics"];
	        this.clearTimes = source["clearTimes"];
	    }
	}
	export class Settings {
	    language: string;
	    overlay: OverlaySettings;
	    hotkeys: Hotkeys;
	    network: NetworkSettings;
	    fps: FPSSettings;
	    sharing: Sharing;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.language = source["language"];
	        this.overlay = this.convertValues(source["overlay"], OverlaySettings);
	        this.hotkeys = this.convertValues(source["hotkeys"], Hotkeys);
	        this.network = this.convertValues(source["network"], NetworkSettings);
	        this.fps = this.convertValues(source["fps"], FPSSettings);
	        this.sharing = this.convertValues(source["sharing"], Sharing);
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

export namespace gamedata {
	
	export class Stats {
	    dataVersion: string;
	    gameVersion: string;
	    characters: number;
	    relics: number;
	    relicsVerified: number;
	    bosses: number;
	    bossesVerified: number;
	    timerProfiles: number;
	    timersVerified: number;
	    priorsVerified: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataVersion = source["dataVersion"];
	        this.gameVersion = source["gameVersion"];
	        this.characters = source["characters"];
	        this.relics = source["relics"];
	        this.relicsVerified = source["relicsVerified"];
	        this.bosses = source["bosses"];
	        this.bossesVerified = source["bossesVerified"];
	        this.timerProfiles = source["timerProfiles"];
	        this.timersVerified = source["timersVerified"];
	        this.priorsVerified = source["priorsVerified"];
	    }
	}

}

export namespace main {
	
	export class AppInfo {
	    version: string;
	    mode: string;
	    configDir: string;
	    data?: gamedata.Stats;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.mode = source["mode"];
	        this.configDir = source["configDir"];
	        this.data = this.convertValues(source["data"], gamedata.Stats);
	        this.error = source["error"];
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

export namespace netmon {
	
	export class Probe {
	    method: string;
	    address: string;
	
	    static createFrom(source: any = {}) {
	        return new Probe(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.address = source["address"];
	    }
	}
	export class Stats {
	    target: string;
	    probe: Probe;
	    lastMs: number;
	    lastLost: boolean;
	    avgMs: number;
	    minMs: number;
	    maxMs: number;
	    jitterMs: number;
	    lossPct: number;
	    sent: number;
	    received: number;
	    totalSent: number;
	    totalLost: number;
	    history: number[];
	    level: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.probe = this.convertValues(source["probe"], Probe);
	        this.lastMs = source["lastMs"];
	        this.lastLost = source["lastLost"];
	        this.avgMs = source["avgMs"];
	        this.minMs = source["minMs"];
	        this.maxMs = source["maxMs"];
	        this.jitterMs = source["jitterMs"];
	        this.lossPct = source["lossPct"];
	        this.sent = source["sent"];
	        this.received = source["received"];
	        this.totalSent = source["totalSent"];
	        this.totalLost = source["totalLost"];
	        this.history = source["history"];
	        this.level = source["level"];
	        this.error = source["error"];
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

