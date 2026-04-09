import { defineStore } from 'pinia'
import { ref } from 'vue'
import { WriteLog, ReadLogs, ClearLogFile } from '../../wailsjs/go/main/App'

export const useLogStore = defineStore('logs', () => {
    const logs = ref([])

    const loadHistoryLogs = async (deviceID) => {
        try {
            const lines = await ReadLogs(deviceID || "")
            if (lines && lines.length > 0) {
                const parsed = []
                for (let line of lines) {
                    try {
                        const obj = JSON.parse(line)
                        // logrus JSON output format: {"device":"COM3","level":"info","msg":"...","time":"2026-03-27 15:42:01"}
                        // Only show logs for this device or system logs
                        if (!deviceID || obj.device === deviceID || obj.device === "System" || obj.device === "") {
                             parsed.unshift({
                                id: Date.now() + Math.random(),
                                time: obj.time,
                                event: obj.level.charAt(0).toUpperCase() + obj.level.slice(1),
                                deviceID: obj.device || "System",
                                details: obj.msg
                            })
                        }
                    } catch(e) {
                        // Fallback to old regex format just in case
                        const match = line.match(/^\[(.*?)\] \[(.*?)\] \[(.*?)\] (.*)$/)
                        if (match) {
                            const [, time, event, devId, details] = match
                            if (!deviceID || devId === deviceID || devId === "System") {
                                parsed.unshift({
                                    id: Date.now() + Math.random(),
                                    time,
                                    event,
                                    deviceID: devId,
                                    details
                                })
                            }
                        }
                    }
                }
                logs.value = parsed
            } else {
                logs.value = []
            }
        } catch(e) {
            console.error("Failed to load logs", e)
        }
    }

    const addLog = (event, details, deviceID) => {
        const time = new Date().toLocaleString()
        const targetDevice = deviceID || "System"
        
        logs.value.unshift({ time, event, deviceID: targetDevice, details })
        if (logs.value.length > 100) logs.value.pop()
        
        WriteLog(targetDevice, event, details)
    }

    const clearLogs = async (deviceID) => {
        try {
            await ClearLogFile(deviceID || "")
            logs.value = []
        } catch (e) {
            console.error("Failed to clear logs", e)
            throw e
        }
    }

    return {
        logs,
        loadHistoryLogs,
        addLog,
        clearLogs
    }
})
