import { defineStore } from 'pinia'
import { ref, reactive, computed } from 'vue'
import { AutoSearchProbe, ClosePort, OpenPort, SendCommand } from '../../wailsjs/go/main/App'
import { parseDevice } from '../utils/deviceParser'

export const useDeviceStore = defineStore('devices', () => {
    const devices = ref([])
    const currentDevice = ref(null)
    const portStates = reactive({})
    const isScanning = ref(false)
    const isBackendConnected = ref(false)

    // Initialize port states
    for (let i = 1; i <= 16; i++) portStates[i] = false

    const devicesFoundCount = computed(() => devices.value.length)
    const selectedDeviceTotalPorts = computed(() => currentDevice.value ? (currentDevice.value.totalPorts || 7) : 7)
    const currentPortObj = computed(() => isBackendConnected.value ? { backend: true } : null)

    const connectViaBackend = async (portPath) => {
        try {
            await OpenPort(portPath)
            isBackendConnected.value = true
        } catch(e) {
            console.error('Connection error:', e)
            isBackendConnected.value = false
        }
    }

    const disconnect = async () => {
        // Disconnect logic is now "Stop managing this device as current"
        // But backend connection remains open.
        // If we want to strictly close it, we can call ClosePort.
        // But the requirement is "keep all devices connected".
        // So we just clear frontend state.
        isBackendConnected.value = false
        currentDevice.value = null
    }

    const autoSearch = async (targetDevId) => {
        isScanning.value = true
        
        // Don't close current port, just reset selection
        if (currentDevice.value) {
            currentDevice.value = null
            isBackendConnected.value = false
        }

        // Reset port states
        for (let i = 1; i <= 16; i++) portStates[i] = false
    
        devices.value = []
        
        try {
            const results = await AutoSearchProbe()
            if (results && results.length > 0) {
                const parsedResults = results.map(parseDevice)
                devices.value = parsedResults
                
                // If targetDevId provided, try to find and select it
                if (targetDevId) {
                    const target = devices.value.find(d => d.portId === targetDevId)
                    if (target) {
                        return target
                    }
                }
            }
        } catch (err) {
            console.error(err)
        } finally {
            isScanning.value = false
        }
        return null
    }

    const selectDevice = async (device) => {
        if (!device) return
        
        // Don't close previous port, just switch context
        isBackendConnected.value = false
        
        // Force update even if selecting same ID, to refresh state
        currentDevice.value = device
        
        // Reset port states
        for (let i = 1; i <= 16; i++) {
            portStates[i] = false
        }
        
        // Connect to new device (OpenPort handles idempotency or reconnect)
        await connectViaBackend(device.portId)
        
        // Query actual status from device
        if (isBackendConnected.value) {
            try {
                // Wait a bit for connection stability and clear any buffer
                await new Promise(r => setTimeout(r, 200))
                
                // Pass deviceID to SendCommand
                const resp = await SendCommand(device.portId, "GP")
                
                if (resp) {
                     // Clean up hexStatus
                     let hexStatus = resp.replace(/[\r\n\s]+/g, '').trim()
                     
                     // Try to find a valid hex pattern
                     // Look for 8 chars first (mask + status), then 2 chars (status byte 0)
                     let match = hexStatus.match(/[0-9A-Fa-f]{8}/)
                     if (!match) match = hexStatus.match(/[0-9A-Fa-f]{2}/)
                     
                     if (match) {
                         const hexVal = match[0]
                         const val = parseInt(hexVal.substring(0, 2), 16)
                         
                         if (!isNaN(val)) {
                             for (let i = 1; i <= 7; i++) {
                                 if ((val >> (i - 1)) & 1) {
                                     portStates[i] = true
                                 } else {
                                     portStates[i] = false
                                 }
                             }
                         }
                     } else {
                        console.warn("Failed to parse GP response:", resp)
                        // Fallback to probe data
                        if (device.onPortsDisplay) {
                            const onPorts = device.onPortsDisplay.split(',').map(p => parseInt(p.trim()))
                            onPorts.forEach(p => {
                                if (p >= 1 && p <= 16) portStates[p] = true
                            })
                        }
                     }
                }
            } catch (e) {
                console.error("Failed to query device status:", e)
                // Fallback to initial probe data
                if (device.onPortsDisplay) {
                    const onPorts = device.onPortsDisplay.split(',').map(p => parseInt(p.trim()))
                    onPorts.forEach(p => {
                        if (p >= 1 && p <= 16) portStates[p] = true
                    })
                }
            }
        } else {
            // Fallback if connection failed
            if (device.onPortsDisplay) {
                const onPorts = device.onPortsDisplay.split(',').map(p => parseInt(p.trim()))
                onPorts.forEach(p => {
                    if (p >= 1 && p <= 16) portStates[p] = true
                })
            }
        }
    }

    const updatePortStatesFromHex = (hexStatus) => {
         // Clean up hexStatus
         let cleanHex = hexStatus.replace(/[\r\n\s]+/g, '').trim()
         
         // Try to find a valid hex pattern
         // Look for 8 chars first (mask + status), then 2 chars (status byte 0)
         let match = cleanHex.match(/[0-9A-Fa-f]{8}/)
         if (!match) match = cleanHex.match(/[0-9A-Fa-f]{2}/)
         
         if (match) {
             const hexVal = match[0]
             const val = parseInt(hexVal.substring(0, 2), 16)
             
             if (!isNaN(val)) {
                 for (let i = 1; i <= 7; i++) {
                     if ((val >> (i - 1)) & 1) {
                         portStates[i] = true
                     } else {
                         portStates[i] = false
                     }
                 }
             }
             return true
         }
         return false
    }

    return {
        devices,
        currentDevice,
        portStates,
        isScanning,
        isBackendConnected,
        devicesFoundCount,
        selectedDeviceTotalPorts,
        currentPortObj,
        autoSearch,
        selectDevice,
        disconnect,
        updatePortStatesFromHex
    }
})
