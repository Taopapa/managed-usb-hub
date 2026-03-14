export namespace config {
	
	export class ScheduledTask {
	    id: string;
	    device_id: string;
	    days_of_week: number[];
	    start_time: string;
	    stop_time: string;
	    start_mask: string;
	    stop_mask: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScheduledTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.device_id = source["device_id"];
	        this.days_of_week = source["days_of_week"];
	        this.start_time = source["start_time"];
	        this.stop_time = source["stop_time"];
	        this.start_mask = source["start_mask"];
	        this.stop_mask = source["stop_mask"];
	        this.enabled = source["enabled"];
	    }
	}

}

export namespace hubmanager {
	
	export class DeviceInfo {
	    path: string;
	    probeResponse: string;
	    asciiResponse: string;
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
	        this.ledStatus = source["ledStatus"];
	        this.goData = source["goData"];
	        this.success = source["success"];
	    }
	}

}

