import { defineStore } from 'pinia'
import { ref, reactive, computed } from 'vue'
import { AutoSearchProbe, ClosePort, GetPortStatus } from '../../wailsjs/go/main/App'
import { parseDevice } from '../utils/deviceParser'
import { CONSTANTS } from '../config/constants'

export const useDeviceStore = defineStore('devices', () => {
    const devices = ref([])
    const currentDevice = ref(null)
    const portStates = reactive({})
    const isScanning = ref(false)

    // Initialize port states
    for (let i = 1; i <= CONSTANTS.MAX_PORTS; i++) portStates[i] = false

    const devicesFoundCount = computed(() => devices.value.length)
    const selectedDeviceTotalPorts = computed(() => 7)

    const disconnect = async () => {
        if (currentDevice.value?.portId) {
            try {
                await ClosePort(currentDevice.value.portId)
            } catch (e) {
                console.error('Failed to close port:', e)
            }
        }
        currentDevice.value = null
    }

    const autoSearch = async (targetDevId) => {
        isScanning.value = true
        
        if (currentDevice.value?.portId) {
            try {
                await ClosePort(currentDevice.value.portId)
            } catch (e) {
                console.error('Failed to close port before scan:', e)
            }
            currentDevice.value = null
        }

        // Reset port states
        for (let i = 1; i <= CONSTANTS.MAX_PORTS; i++) portStates[i] = false
    
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

        if (currentDevice.value?.portId && currentDevice.value.portId !== device.portId) {
            try {
                await ClosePort(currentDevice.value.portId)
            } catch (e) {
                console.error('Failed to close previous port:', e)
            }
        }

        currentDevice.value = device

        for (let i = 1; i <= CONSTANTS.MAX_PORTS; i++) {
            portStates[i] = false
        }

        try {
            const statesMap = await GetPortStatus(device.portId, selectedDeviceTotalPorts.value)
            if (statesMap) {
                for (let i = 1; i <= selectedDeviceTotalPorts.value; i++) {
                    portStates[i] = !!statesMap[i]
                }
                return
            }
        } catch (e) {
            console.error("Failed to query device status:", e)
        }

        if (device.onPortsDisplay) {
            const onPorts = device.onPortsDisplay.split(',').map(p => parseInt(p.trim()))
            onPorts.forEach(p => {
                if (p >= 1 && p <= CONSTANTS.MAX_PORTS) portStates[p] = true
            })
        }
    }

    const updatePortStatesFromHex = (deviceId, hexResponse) => {
        // Find device
        const device = devices.value.find(d => d.portId === deviceId)
        if (!device) return

        let cleanHex = hexResponse.trim()
        if (cleanHex.toUpperCase().startsWith("GW")) cleanHex = cleanHex.substring(2)
        else if (cleanHex.startsWith("G") || cleanHex.startsWith("g")) cleanHex = cleanHex.substring(1)
        
        // Parse first byte (port mask)
        let maskByte = 0
        if (cleanHex.length >= 2) {
            maskByte = parseInt(cleanHex.substring(0, 2), 16)
        } else {
            maskByte = parseInt(cleanHex, 16)
        }

        if (isNaN(maskByte)) return

        // Update portStates reactive object if this is the currently selected device
        if (currentDevice.value && currentDevice.value.portId === deviceId) {
            for (let i = 1; i <= selectedDeviceTotalPorts.value; i++) {
                portStates[i] = ((maskByte >> (i - 1)) & 1) !== 0
            }
        }

        // Update port states in the device list
        // Bit 0 = Port 1, Bit 1 = Port 2, etc.
        for (let i = 0; i < selectedDeviceTotalPorts.value; i++) {
            const isOn = (maskByte & (1 << i)) !== 0
            if (device.ports[i]) {
                device.ports[i].state = isOn
            }
        }
    }

    return {
        devices,
        currentDevice,
        portStates,
        isScanning,
        devicesFoundCount,
        selectedDeviceTotalPorts,
        autoSearch,
        selectDevice,
        disconnect,
        updatePortStatesFromHex
    }
})
