<script setup>
import { inject } from 'vue'
import { SendCommand } from '../../wailsjs/go/main/App'
import { useDeviceStore } from '../stores/devices'
import { useAuthStore } from '../stores/auth'
import { useLogStore } from '../stores/logs'
import { useUIStore } from '../stores/ui'
import { storeToRefs } from 'pinia'

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
const { executeWithAuth } = authStore
const { showSetPasswordModal, setPassOld } = storeToRefs(authStore)
const { showAlert } = uiStore

const togglePort = (n) => {
    if (!currentDevice.value) return
    if (n > selectedDeviceTotalPorts.value) return
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

const sendSPCommand = async (newPortStates, logDescription, retryCallback) => {
    let password = currentDevice.value ? currentDevice.value.sessionPassword : null
    
    if (!currentDevice.value || !password) {
        return
    }

    const statesToUse = newPortStates || portStates
    const total = selectedDeviceTotalPorts.value
    const onCount = Object.values(statesToUse).filter(v => v).length
    let maskHex = ""

    if (onCount === total && total > 0) {
        maskHex = "FFFFFFFF"
    } else if (onCount === 0) {
        maskHex = "00000000"
    } else {
        let byte0 = 0x80
        for (let i = 1; i <= 7; i++) {
            if (statesToUse[i]) byte0 |= (1 << (i - 1))
        }
        maskHex = byte0.toString(16).toUpperCase().padStart(2, '0') + "FFFFFF"
    }

    const cmd = `SP${password}${maskHex}`
    try {
        const resp = await SendCommand(currentDevice.value.portId, cmd)
        console.log(resp)
        if (resp && resp.includes('E01')) {
             currentDevice.value.sessionPassword = null
             // Trigger re-auth and retry via callback if provided
             if (retryCallback) {
                 // Force auth because we know current password failed
                 executeWithAuth(async () => {
                     retryCallback()
                 }, null, true)
             } else {
                 // Fallback if no callback
                 executeWithAuth(async () => {
                     await sendSPCommand(newPortStates, logDescription)
                 }, null, true)
             }
             return
        } else if (resp && resp.startsWith('G')) {
             // Update local state to match
             for(let i=1; i<=16; i++) portStates[i] = statesToUse[i]
             addLog('Command', logDescription || 'Port states updated', currentDevice.value.portId)
        }
    } catch(e) {
        addLog('Error', 'Command failed: ' + e, currentDevice.value.portId)
    }
}

const allOn = () => {
    executeWithAuth(async () => {
        const newStates = { ...portStates }
        const total = selectedDeviceTotalPorts.value
        for (let i = 1; i <= total; i++) newStates[i] = true
        
        const performAllOn = async () => {
            await sendSPCommand(newStates, "All Ports Enabled", performAllOn)
        }
        await performAllOn()
    })
}

const allOff = () => {
    executeWithAuth(async () => {
        const newStates = { ...portStates }
        const total = selectedDeviceTotalPorts.value
        for (let i = 1; i <= total; i++) newStates[i] = false
        
        const performAllOff = async () => {
            await sendSPCommand(newStates, "All Ports Disabled", performAllOff)
        }
        await performAllOff()
    })
}

const setPasswordAction = async () => {
    executeWithAuth(async () => {
        setPassOld.value = currentDevice.value.sessionPassword || "pass    "
        showSetPasswordModal.value = true
    })
}

const resetHubAction = async () => {
    executeWithAuth(async () => {
        let password = currentDevice.value.sessionPassword
        const cmd = `RH${password}`
        try {
            const resp = await SendCommand(currentDevice.value.portId, cmd)
            
            if (resp && resp.includes('E01')) {
                 currentDevice.value.sessionPassword = null
                 resetHubAction()
                 return
            }
            
            addLog('User Action', 'Reset Hub command sent', currentDevice.value.portId)
            
            // Silent success, but trigger reconnect flow
            
            const devId = currentDevice.value.portId
            await disconnect()
            currentDevice.value = null
            
            // Auto-refresh after 3s
            setTimeout(async () => {
                const foundDev = await autoSearch(devId)
                if (foundDev) {
                    await selectDevice(foundDev)
                }
            }, 3000)
        } catch(e) {
            if(showAlert) showAlert('Failed to reset hub: ' + e, "Error")
        }
    })
}

const restoreDefaultAction = async () => {
    executeWithAuth(async () => {
        let password = currentDevice.value.sessionPassword || "pass    "
        const cmd = `RD${password}`
        try {
            const resp = await SendCommand(currentDevice.value.portId, cmd)
            
            if (resp && resp.includes('E01')) {
                 currentDevice.value.sessionPassword = null
                 restoreDefaultAction()
                 return
            }
            
            addLog('User Action', 'Restored to default settings', currentDevice.value.portId)
            
            const dev = currentDevice.value
            await disconnect()
            currentDevice.value = null
            
            await new Promise(r => setTimeout(r, 1000))
            await selectDevice(dev)
            
            for(let i=1; i<=16; i++) portStates[i] = true
            
            // Silent success
            // if(showAlert) showAlert('Device restored to defaults and reconnected.', "Success")
            
        } catch (e) {
             addLog('Error', 'Restore Default failed: ' + e, currentDevice.value ? currentDevice.value.portId : 'System')
             if(showAlert) showAlert('Restore Default failed: ' + e, "Error")
        }
    })
}

const savePortStates = async () => {
    executeWithAuth(async () => {
        let password = currentDevice.value.sessionPassword
        const cmd = `WP${password}`
        try {
            const resp = await SendCommand(currentDevice.value.portId, cmd)
            
            if (resp && resp.startsWith('G')) {
                addLog('User Action', 'Port states saved to device memory', currentDevice.value.portId)
                
                const dev = currentDevice.value
                await disconnect()
                currentDevice.value = null

                await new Promise(r => setTimeout(r, 1000))
                
                await autoSearch()
                const foundDev = devices.value.find(d => d.portId === dev.portId)
                if (foundDev) {
                    await selectDevice(foundDev)
                }
                
                // Silent success
                // if(showAlert) showAlert('Port states saved successfully and device reconnected.', "Success")
            } else if (resp && resp.includes('E01')) {
                 currentDevice.value.sessionPassword = null
                 savePortStates()
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
                <button @click="allOn" :disabled="!currentDevice">Enable All</button>
                <button @click="allOff" :disabled="!currentDevice">Disable All</button>
            </div>
            
            <div class="ports-visual">
                <div v-for="n in 7" :key="n" 
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
                <button @click="setPasswordAction" :disabled="!currentDevice">Set Password</button>
                <button @click="resetHubAction" :disabled="!currentDevice">Reset Hub</button>
                <button @click="restoreDefaultAction" :disabled="!currentDevice">Restore Default</button>
                <button @click="savePortStates" :disabled="!currentDevice">Save Port States</button>
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
