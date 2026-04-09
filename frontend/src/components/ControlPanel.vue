<script setup>
import { ref } from 'vue'
import { SetPortStatus, ResetHub, RestoreDefault, SavePortStates } from '../../wailsjs/go/main/App'
import { useDeviceStore } from '../stores/devices'
import { useAuthStore } from '../stores/auth'
import { useLogStore } from '../stores/logs'
import { useUIStore } from '../stores/ui'
import { storeToRefs } from 'pinia'
import { CONSTANTS } from '../config/constants'

import connectedIcon from '../static/connected.png'
import disconnectedIcon from '../static/disconnect.png'

const deviceStore = useDeviceStore()
const authStore = useAuthStore()
const logStore = useLogStore()
const uiStore = useUIStore()

const { currentDevice, selectedDeviceTotalPorts, devices } = storeToRefs(deviceStore)
const { portStates } = deviceStore // direct access to reactive object
const { autoSearch, selectDevice, disconnect } = deviceStore
const { addLog } = logStore
const { executeWithAuth, markPasswordRejected, clearSessionPassword } = authStore
const { showSetPasswordModal, setPassOld } = storeToRefs(authStore)
const { showAlert, showConfirm } = uiStore
const isPortCommandPending = ref(false)

const reconnectDeviceAfterCommand = async (devId, delayMs = 3000, options = {}) => {
    const { resetToAllOn = false, clearPassword = false } = options

    if (clearPassword) {
        await clearSessionPassword(devId)
    }

    await disconnect()
    currentDevice.value = null

    await new Promise(r => setTimeout(r, delayMs))

    const foundDev = await autoSearch(devId)
    if (foundDev) {
        await selectDevice(foundDev)
    } else if (devices.value.length === 1) {
        await selectDevice(devices.value[0])
    } else {
        addLog('Warn', 'Device not found after command; please scan and reconnect.', devId)
        return
    }

    if (resetToAllOn) {
        for (let i = 1; i <= 16; i++) portStates[i] = true
    }
}

const togglePort = (n) => {
    if (!currentDevice.value) return
    if (n > selectedDeviceTotalPorts.value) return
    if (isPortCommandPending.value) return
    executeWithAuth(async () => {
        const newStates = { ...portStates }
        newStates[n] = !newStates[n]
        const desc = `Port ${n} ${newStates[n] ? 'Enabled' : 'Disabled'}`
        
        // Use a wrapper function to handle recursion
        const performToggle = async () => {
            await sendSPCommand(newStates, desc, performToggle)
        }
        await performToggle()
    })
}

const sendSPCommand = async (newPortStates, logDescription, retryCallback, attempt = 0) => {
    let password = currentDevice.value ? (currentDevice.value.sessionPassword || CONSTANTS.DEFAULT_PASSWORD) : null
    
    if (!currentDevice.value || !password) {
        return
    }

    const statesToUse = newPortStates || portStates
    if (isPortCommandPending.value) {
        return
    }
    isPortCommandPending.value = true
    try {
        const resp = await SetPortStatus(currentDevice.value.portId, password, statesToUse, selectedDeviceTotalPorts.value)
        console.log(resp)
        const normalizedResp = String(resp || '').trim()
        const looksLikeStateResponse = normalizedResp.startsWith('G') || /^[0-9A-Fa-f]{2,8}$/.test(normalizedResp)
        if (!resp || !String(resp).trim()) {
             addLog('Error', 'No response from device for SetPortStatus', currentDevice.value.portId)
             if(showAlert) showAlert('Device did not respond to the port command.', "Communication Error")
             return
        }
        if (resp && resp.includes('E01')) {
             await markPasswordRejected(currentDevice.value.portId)
             
             if (attempt >= CONSTANTS.MAX_AUTH_RETRIES) {
                 // Stop recursion and show error
                 addLog('Error', 'Authentication failed after retry', currentDevice.value.portId)
                 if(showAlert) showAlert('Device authentication failed. It may be locked or communication is disrupted.', "Error")
                 return
             }
             
             // Trigger re-auth and retry via callback if provided
             if (retryCallback) {
                 // Force auth because we know current password failed
                 executeWithAuth(async () => {
                     // Since executeWithAuth resolves when auth succeeds, we now call the retry logic.
                     await retryCallback(attempt + 1)
                 }, null, true)
             } else {
                 // Fallback if no callback
                 executeWithAuth(async () => {
                     await sendSPCommand(newPortStates, logDescription, null, attempt + 1)
                 }, null, true)
             }
             return
        } else if (looksLikeStateResponse) {
             // Update local state to match
             for(let i=1; i<=CONSTANTS.MAX_PORTS; i++) portStates[i] = !!statesToUse[i]
             addLog('Command', logDescription || 'Port states updated', currentDevice.value.portId)
        } else {
             addLog('Error', `Unexpected response for SetPortStatus: ${normalizedResp}`, currentDevice.value.portId)
             if(showAlert) showAlert(`Unexpected device response: ${normalizedResp}`, "Communication Error")
        }
    } catch(e) {
        addLog('Error', 'Command failed: ' + e, currentDevice.value.portId)
        if(showAlert) showAlert('Port command failed: ' + e, "Error")
    } finally {
        isPortCommandPending.value = false
    }
}

const allOn = () => {
    if (isPortCommandPending.value) return
    executeWithAuth(async () => {
        const newStates = { ...portStates }
        const total = selectedDeviceTotalPorts.value
        for (let i = 1; i <= total; i++) newStates[i] = true
        
        const performAllOn = async (attempt = 0) => {
            await sendSPCommand(newStates, "All Ports Enabled", performAllOn, attempt)
        }
        await performAllOn()
    })
}

const allOff = () => {
    if (isPortCommandPending.value) return
    executeWithAuth(async () => {
        const newStates = { ...portStates }
        const total = selectedDeviceTotalPorts.value
        for (let i = 1; i <= total; i++) newStates[i] = false
        
        const performAllOff = async (attempt = 0) => {
            await sendSPCommand(newStates, "All Ports Disabled", performAllOff, attempt)
        }
        await performAllOff()
    })
}

const setPasswordAction = async () => {
    executeWithAuth(async () => {
        setPassOld.value = currentDevice.value.sessionPassword || CONSTANTS.DEFAULT_PASSWORD
        showSetPasswordModal.value = true
    })
}

const resetHubAction = async (attempt = 0) => {
    // Only prompt for confirmation on the first attempt
    if (attempt === 0) {
        const confirmed = await showConfirm(`Are you sure you want to reset the hub? This will disrupt all connections temporarily.`, 'Confirm Reset Hub')
        if (!confirmed) return
    }

    executeWithAuth(async () => {
        let password = currentDevice.value.sessionPassword || CONSTANTS.DEFAULT_PASSWORD
        try {
            const resp = await ResetHub(currentDevice.value.portId, password)
            
            if (resp && resp.includes('E01')) {
                 await markPasswordRejected(currentDevice.value.portId)
                 if (attempt >= CONSTANTS.MAX_AUTH_RETRIES) {
                     addLog('Error', 'Reset Hub authentication failed', currentDevice.value.portId)
                     if(showAlert) showAlert('Device authentication failed. Cannot reset hub.', "Error")
                     return
                 }
                 executeWithAuth(async () => {
                     await resetHubAction(attempt + 1)
                 }, null, true)
                 return
            }
            
            addLog('User Action', 'Reset Hub command sent', currentDevice.value.portId)
            
            const devId = currentDevice.value.portId
            setTimeout(async () => {
                await reconnectDeviceAfterCommand(devId, 3000)
                
                // After reconnecting, actively fetch the real device name
                executeWithAuth(async () => {
                    try {
                        await authStore.handlePasswordSubmit(password)
                    } catch(e) {}
                }, null, false)
            }, 3000)
        } catch(e) {
            if(showAlert) showAlert('Failed to reset hub: ' + e, "Error")
        }
    })
}

const restoreDefaultAction = async (attempt = 0) => {
    if (attempt === 0) {
        const confirmed = await showConfirm(`Are you sure you want to restore the hub to factory defaults? All settings will be lost.`, 'Confirm Restore Default')
        if (!confirmed) return
    }

    executeWithAuth(async () => {
        let password = currentDevice.value.sessionPassword || CONSTANTS.DEFAULT_PASSWORD
        try {
            const resp = await RestoreDefault(currentDevice.value.portId, password)
            
            if (resp && resp.includes('E01')) {
                 await markPasswordRejected(currentDevice.value.portId)
                 if (attempt >= CONSTANTS.MAX_AUTH_RETRIES) {
                     addLog('Error', 'Restore Default authentication failed', currentDevice.value.portId)
                     if(showAlert) showAlert('Device authentication failed. Cannot restore defaults.', "Error")
                     return
                 }
                 executeWithAuth(async () => {
                     await restoreDefaultAction(attempt + 1)
                 }, null, true)
                 return
            }
            
            addLog('User Action', 'Restored to default settings', currentDevice.value.portId)
            
            const devId = currentDevice.value.portId
            await reconnectDeviceAfterCommand(devId, 3000, { resetToAllOn: true, clearPassword: true })
            
            // After reconnecting, actively fetch the real device name
            executeWithAuth(async () => {
                try {
                    // Trigger a name refresh behind the scenes using the current (default) password
                    await authStore.handlePasswordSubmit(CONSTANTS.DEFAULT_PASSWORD)
                } catch(e) {}
            }, null, false)
            
        } catch (e) {
             addLog('Error', 'Restore Default failed: ' + e, currentDevice.value ? currentDevice.value.portId : 'System')
             if(showAlert) showAlert('Restore Default failed: ' + e, "Error")
        }
    })
}

const savePortStates = async (attempt = 0) => {
    if (attempt === 0) {
        const confirmed = await showConfirm(`Are you sure you want to save current port states to device memory? They will be applied on the next boot.`, 'Confirm Save Port States')
        if (!confirmed) return
    }

    executeWithAuth(async () => {
        let password = currentDevice.value.sessionPassword || CONSTANTS.DEFAULT_PASSWORD
        try {
            const resp = await SavePortStates(currentDevice.value.portId, password)
            
            if (resp && resp.startsWith('G')) {
                addLog('User Action', 'Port states saved to device memory', currentDevice.value.portId)
                
                const devId = currentDevice.value.portId
                await reconnectDeviceAfterCommand(devId, 1000)
                
                executeWithAuth(async () => {
                    try {
                        await authStore.handlePasswordSubmit(password)
                    } catch(e) {}
                }, null, false)
            } else if (resp && resp.includes('E01')) {
                 await markPasswordRejected(currentDevice.value.portId)
                 if (attempt >= CONSTANTS.MAX_AUTH_RETRIES) {
                     addLog('Error', 'Save Port States authentication failed', currentDevice.value.portId)
                     if(showAlert) showAlert('Device authentication failed. Cannot save states.', "Error")
                     return
                 }
                 executeWithAuth(async () => {
                     await savePortStates(attempt + 1)
                 }, null, true)
            } else {
                 addLog('Error', `Save Port States failed: ${resp}`, currentDevice.value.portId)
                 if(showAlert) showAlert(`Failed to save port states. Device responded: ${resp}`, "Error")
            }
        } catch(e) {
            addLog('Error', 'Save Port States command failed: ' + e, currentDevice.value.portId)
            if(showAlert) showAlert('Error saving port states: ' + e, "Error")
        }
    })
}
</script>

<template>
    <div class="control-panel">
        <div class="control-label">Click a port to toggle state</div>
        <div class="control-row">
            <div class="left-buttons">
                <button @click="allOn" :disabled="!currentDevice || isPortCommandPending">Enable All</button>
                <button @click="allOff" :disabled="!currentDevice || isPortCommandPending">Disable All</button>
            </div>
            
            <div class="ports-visual">
                <div v-for="n in selectedDeviceTotalPorts" :key="n" 
                     class="port-icon" 
                     :class="{ 'enabled': portStates[n], 'disabled': !portStates[n] }"
                     @click="togglePort(n)">
                    <div class="port-arrow">
                         <img v-if="portStates[n]" :src="connectedIcon" width="40" height="40" style="object-fit: contain;" alt="Connected" />
                         <img v-else :src="disconnectedIcon" width="40" height="40" style="object-fit: contain;" alt="Disconnected" />
                    </div>
                    <span class="port-number">{{ n }}</span>
                </div>
            </div>

            <div class="right-buttons">
                <button @click="setPasswordAction" :disabled="!currentDevice || isPortCommandPending">Set Password</button>
                <button @click="resetHubAction" :disabled="!currentDevice || isPortCommandPending">Reset Hub</button>
                <button @click="restoreDefaultAction" :disabled="!currentDevice || isPortCommandPending">Restore Default</button>
                <button @click="savePortStates" :disabled="!currentDevice || isPortCommandPending">Save Port States</button>
            </div>
        </div>
    </div>
</template>

<style scoped>
.control-panel {
    background-color: #e0e0e0;
    padding: 10px;
    border: 1px solid #ccc;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 5px;
}

.control-label {
    font-weight: bold;
    margin-bottom: 5px;
}

.control-row {
    display: flex;
    justify-content: space-between;
    width: 100%;
    align-items: center;
}

.left-buttons, .right-buttons {
    display: flex;
    justify-content: center;
    flex-wrap: wrap;
    gap: 5px;
}

.left-buttons button, .right-buttons button {
    width: 120px;
    padding: 4px;
}

.ports-visual {
    display: flex;
    gap: 5px;
    background-color: #d0d0d0;
    padding: 5px;
    border-radius: 4px;
}

.port-icon {
    display: flex;
    flex-direction: column;
    align-items: center;
    cursor: pointer;
    padding: 2px;
    border: 1px solid transparent;
}

.port-icon:hover {
    background-color: #fff;
    border-color: #999;
}

.port-arrow {
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
}

.port-number {
    font-weight: bold;
}

button {
    cursor: pointer;
    background-color: #e1e1e1;
    border: 1px solid #adadad;
    padding: 2px 10px;
}

button:hover {
    background-color: #e5f1fb;
    border-color: #0078d7;
}

button:disabled {
    color: #838383;
    background-color: #f0f0f0;
    border-color: #d0d0d0;
}
</style>
