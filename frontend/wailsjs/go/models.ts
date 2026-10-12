export namespace boss {
	
	export class Candidate {
	    nightlord: string;
	    probability: number;
	    excluded: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nightlord = source["nightlord"];
	        this.probability = source["probability"];
	        this.excluded = source["excluded"];
	    }
	}
	export class Chance {
	    boss: string;
	    probability: number;
	
	    static createFrom(source: any = {}) {
	        return new Chance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.boss = source["boss"];
	        this.probability = source["probability"];
	    }
	}
	export class Query {
	    night1: string;
	    night2: string;
	    exclude: string[];
	
	    static createFrom(source: any = {}) {
	        return new Query(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.night1 = source["night1"];
	        this.night2 = source["night2"];
	        this.exclude = source["exclude"];
	    }
	}
	export class Result {
	    ranking: Candidate[];
	    signal: boolean;
	    impossible: boolean;
	    night1: Chance[];
	    night2: Chance[];
	    runs: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ranking = this.convertValues(source["ranking"], Candidate);
	        this.signal = source["signal"];
	        this.impossible = source["impossible"];
	        this.night1 = this.convertValues(source["night1"], Chance);
	        this.night2 = this.convertValues(source["night2"], Chance);
	        this.runs = source["runs"];
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
	export class TimerSettings {
	    profile: string;
	    sound: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TimerSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profile = source["profile"];
	        this.sound = source["sound"];
	    }
	}
	export class Settings {
	    language: string;
	    overlay: OverlaySettings;
	    hotkeys: Hotkeys;
	    network: NetworkSettings;
	    fps: FPSSettings;
	    timer: TimerSettings;
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
	        this.timer = this.convertValues(source["timer"], TimerSettings);
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
	
	export class BossForm {
	    name: string;
	    gameIds: number[];
	    health: number[];
	    poise: number[];
	    negation: Record<string, number>;
	    resist: Record<string, number>;
	    immune: string[];
	
	    static createFrom(source: any = {}) {
	        return new BossForm(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.gameIds = source["gameIds"];
	        this.health = source["health"];
	        this.poise = source["poise"];
	        this.negation = source["negation"];
	        this.resist = source["resist"];
	        this.immune = source["immune"];
	    }
	}
	export class BossStats {
	    normal: BossForm[];
	    everdark: BossForm[];
	
	    static createFrom(source: any = {}) {
	        return new BossStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.normal = this.convertValues(source["normal"], BossForm);
	        this.everdark = this.convertValues(source["everdark"], BossForm);
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
	export class Localized {
	    en: string;
	    vi: string;
	
	    static createFrom(source: any = {}) {
	        return new Localized(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.en = source["en"];
	        this.vi = source["vi"];
	    }
	}
	export class Boss {
	    id: string;
	    kind: string;
	    name: Localized;
	    expedition?: Localized;
	    night?: number;
	    dlc?: boolean;
	    stats?: BossStats;
	    verified: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Boss(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.name = this.convertValues(source["name"], Localized);
	        this.expedition = this.convertValues(source["expedition"], Localized);
	        this.night = source["night"];
	        this.dlc = source["dlc"];
	        this.stats = this.convertValues(source["stats"], BossStats);
	        this.verified = source["verified"];
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
	
	export class BossPool {
	    nightlord: string;
	    night1: string[];
	    night2: string[];
	
	    static createFrom(source: any = {}) {
	        return new BossPool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nightlord = source["nightlord"];
	        this.night1 = source["night1"];
	        this.night2 = source["night2"];
	    }
	}
	
	export class Character {
	    id: string;
	    name: Localized;
	    dlc?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Character(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = this.convertValues(source["name"], Localized);
	        this.dlc = source["dlc"];
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
	
	export class Phase {
	    id: string;
	    kind: string;
	    seconds: number;
	    label: Localized;
	
	    static createFrom(source: any = {}) {
	        return new Phase(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.seconds = source["seconds"];
	        this.label = this.convertValues(source["label"], Localized);
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
	export class RelicEffect {
	    id: string;
	    kind: string;
	    categories: string[];
	    name: Localized;
	    inGame?: string;
	    effect?: Localized;
	    stacking: string;
	    notes?: Localized;
	    character?: string;
	    verified: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RelicEffect(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.categories = source["categories"];
	        this.name = this.convertValues(source["name"], Localized);
	        this.inGame = source["inGame"];
	        this.effect = this.convertValues(source["effect"], Localized);
	        this.stacking = source["stacking"];
	        this.notes = this.convertValues(source["notes"], Localized);
	        this.character = source["character"];
	        this.verified = source["verified"];
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
	export class TimerProfile {
	    id: string;
	    name: Localized;
	    verified: boolean;
	    note?: string;
	    phases: Phase[];
	
	    static createFrom(source: any = {}) {
	        return new TimerProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = this.convertValues(source["name"], Localized);
	        this.verified = source["verified"];
	        this.note = source["note"];
	        this.phases = this.convertValues(source["phases"], Phase);
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

export namespace gamenet {
	
	export class Flow {
	    proto: string;
	    remote: string;
	    ip: string;
	    kind: string;
	    pktsInPerSec: number;
	    pktsOutPerSec: number;
	    kbpsIn: number;
	    kbpsOut: number;
	    maxGapMs: number;
	    sendOnly?: boolean;
	    idleMs: number;
	    ageSec: number;
	    via?: string;
	    ping?: netmon.Stats;
	    pingAddr?: string;
	
	    static createFrom(source: any = {}) {
	        return new Flow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proto = source["proto"];
	        this.remote = source["remote"];
	        this.ip = source["ip"];
	        this.kind = source["kind"];
	        this.pktsInPerSec = source["pktsInPerSec"];
	        this.pktsOutPerSec = source["pktsOutPerSec"];
	        this.kbpsIn = source["kbpsIn"];
	        this.kbpsOut = source["kbpsOut"];
	        this.maxGapMs = source["maxGapMs"];
	        this.sendOnly = source["sendOnly"];
	        this.idleMs = source["idleMs"];
	        this.ageSec = source["ageSec"];
	        this.via = source["via"];
	        this.ping = this.convertValues(source["ping"], netmon.Stats);
	        this.pingAddr = source["pingAddr"];
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
	export class Status {
	    state: string;
	    pid: number;
	    flows: Flow[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.pid = source["pid"];
	        this.flows = this.convertValues(source["flows"], Flow);
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
	export class BossData {
	    bosses: gamedata.Boss[];
	    pools: gamedata.BossPool[];
	    characters: gamedata.Character[];
	    poolsVerified: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BossData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bosses = this.convertValues(source["bosses"], gamedata.Boss);
	        this.pools = this.convertValues(source["pools"], gamedata.BossPool);
	        this.characters = this.convertValues(source["characters"], gamedata.Character);
	        this.poolsVerified = source["poolsVerified"];
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
	export class RelicData {
	    effects: gamedata.RelicEffect[];
	    characters: gamedata.Character[];
	
	    static createFrom(source: any = {}) {
	        return new RelicData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.effects = this.convertValues(source["effects"], gamedata.RelicEffect);
	        this.characters = this.convertValues(source["characters"], gamedata.Character);
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

export namespace store {
	
	export class Run {
	    id: string;
	    // Go type: time
	    createdAt: any;
	    mode: string;
	    depth: number;
	    character: string;
	    night1Boss: string;
	    night2Boss: string;
	    shiftingEarth: string;
	    nightlord: string;
	    result: string;
	    clearSeconds: number;
	    buildJson: string;
	
	    static createFrom(source: any = {}) {
	        return new Run(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.mode = source["mode"];
	        this.depth = source["depth"];
	        this.character = source["character"];
	        this.night1Boss = source["night1Boss"];
	        this.night2Boss = source["night2Boss"];
	        this.shiftingEarth = source["shiftingEarth"];
	        this.nightlord = source["nightlord"];
	        this.result = source["result"];
	        this.clearSeconds = source["clearSeconds"];
	        this.buildJson = source["buildJson"];
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

export namespace timer {
	
	export class Snapshot {
	    profile: string;
	    verified: boolean;
	    state: string;
	    phase: number;
	    phaseId: string;
	    kind: string;
	    label: gamedata.Localized;
	    nextKind: string;
	    nextLabel: gamedata.Localized;
	    duration: number;
	    remaining: number;
	    phaseElapsed: number;
	    elapsed: number;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profile = source["profile"];
	        this.verified = source["verified"];
	        this.state = source["state"];
	        this.phase = source["phase"];
	        this.phaseId = source["phaseId"];
	        this.kind = source["kind"];
	        this.label = this.convertValues(source["label"], gamedata.Localized);
	        this.nextKind = source["nextKind"];
	        this.nextLabel = this.convertValues(source["nextLabel"], gamedata.Localized);
	        this.duration = source["duration"];
	        this.remaining = source["remaining"];
	        this.phaseElapsed = source["phaseElapsed"];
	        this.elapsed = source["elapsed"];
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

