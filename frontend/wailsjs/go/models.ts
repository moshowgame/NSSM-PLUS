export namespace service {
	
	export class FieldDiff {
	    field: string;
	    fileValue: string;
	    actualValue: string;
	
	    static createFrom(source: any = {}) {
	        return new FieldDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.fileValue = source["fileValue"];
	        this.actualValue = source["actualValue"];
	    }
	}
	export class ServiceConfig {
	    serviceName: string;
	    displayName: string;
	    description: string;
	    appPath: string;
	    arguments: string;
	    workDir: string;
	    startType: string;
	    account: string;
	    password: string;
	    environment: Record<string, string>;
	    logStdout: string;
	    logStderr: string;
	    rotateLog: boolean;
	    restartDelay: number;
	    restartTimeout: number;
	    dependencies: string[];
	
	    static createFrom(source: any = {}) {
	        return new ServiceConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serviceName = source["serviceName"];
	        this.displayName = source["displayName"];
	        this.description = source["description"];
	        this.appPath = source["appPath"];
	        this.arguments = source["arguments"];
	        this.workDir = source["workDir"];
	        this.startType = source["startType"];
	        this.account = source["account"];
	        this.password = source["password"];
	        this.environment = source["environment"];
	        this.logStdout = source["logStdout"];
	        this.logStderr = source["logStderr"];
	        this.rotateLog = source["rotateLog"];
	        this.restartDelay = source["restartDelay"];
	        this.restartTimeout = source["restartTimeout"];
	        this.dependencies = source["dependencies"];
	    }
	}
	export class ServiceDiff {
	    serviceName: string;
	    state: string;
	    fields?: FieldDiff[];
	
	    static createFrom(source: any = {}) {
	        return new ServiceDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serviceName = source["serviceName"];
	        this.state = source["state"];
	        this.fields = this.convertValues(source["fields"], FieldDiff);
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
	export class ServiceInfo {
	    name: string;
	    displayName: string;
	    status: string;
	    startType: string;
	    appPath: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.status = source["status"];
	        this.startType = source["startType"];
	        this.appPath = source["appPath"];
	    }
	}

}

