export namespace hubmanager {
	
	export class DeviceInfo {
	    path: string;
	    probeResponse: string;
	    asciiResponse: string;
	    deviceName: string;
	    deviceUid: string;
	    ledStatus: string;
	    goData: string;
	    success: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeviceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.probeResponse = source["probeResponse"];
	        this.asciiResponse = source["asciiResponse"];
	        this.deviceName = source["deviceName"];
	        this.deviceUid = source["deviceUid"];
	        this.ledStatus = source["ledStatus"];
	        this.goData = source["goData"];
	        this.success = source["success"];
	    }
	}

}

