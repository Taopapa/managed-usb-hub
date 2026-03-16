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
                const regexNew = /^\[(.*?)\] \[(.*?)\] \[(.*?)\] (.*)$/
                
                lines.forEach(line => {
                    let match = line.match(regexNew)
                    if (match) {
                        parsed.unshift({
                            time: match[1],
                            event: match[2],
                            deviceID: match[3],
                            details: match[4]
                        })
                    }
                })
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
