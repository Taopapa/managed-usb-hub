import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { SetStoredPassword, ChangePassword, GetDeviceName } from '../../wailsjs/go/main/App'
import { useDeviceStore } from './devices'
import { useLogStore } from './logs'
import { useUIStore } from './ui'
import { CONSTANTS } from '../config/constants'

export const useAuthStore = defineStore('auth', () => {
    const deviceStore = useDeviceStore()
    const logStore = useLogStore()
    const uiStore = useUIStore()

    const showPasswordModal = ref(false)
    const authPromptPassword = ref('')
    const showSetPasswordModal = ref(false)
    const setPassOld = ref('')
    
    // In-memory cache for session passwords per port
    const sessionPasswords = reactive(new Map()) 
    const authRequiredPorts = new Set()
    let pendingAction = null

    const getPreferredPassword = (portId) => {
        if (!portId) return CONSTANTS.DEFAULT_PASSWORD

        if (deviceStore.currentDevice?.portId === portId && deviceStore.currentDevice.sessionPassword) {
            return deviceStore.currentDevice.sessionPassword
        }

        return sessionPasswords.get(portId) || CONSTANTS.DEFAULT_PASSWORD
    }

    const markPasswordRejected = async (portId) => {
        if (!portId) return

        authRequiredPorts.add(portId)
        sessionPasswords.delete(portId)

        if (deviceStore.currentDevice?.portId === portId) {
            deviceStore.currentDevice.sessionPassword = ''
        }

        authPromptPassword.value = CONSTANTS.DEFAULT_PASSWORD
        await SetStoredPassword(portId, "")
    }

    const executeWithAuth = async (action, alertFn, forceAuth = false) => {
        const alertFunc = alertFn || uiStore.showAlert
        if (!deviceStore.currentDevice || !deviceStore.currentDevice.portId) {
            if(alertFunc) alertFunc('Please connect a device first.', 'Connection Required')
            return
        }
        const portId = deviceStore.currentDevice.portId
        const preferredPassword = getPreferredPassword(portId)
        const hasActivePassword = !!deviceStore.currentDevice.sessionPassword
        
        if (!forceAuth) {
            if (hasActivePassword) {
                action()
                return
            }

            if (!authRequiredPorts.has(portId)) {
                deviceStore.currentDevice.sessionPassword = preferredPassword
                action()
                return
            }
        }
    
        pendingAction = action
        authPromptPassword.value = preferredPassword
        showPasswordModal.value = true
    }

    const handlePasswordSubmit = async (password) => {
        if (deviceStore.currentDevice) {
            const portId = deviceStore.currentDevice.portId
            const paddedPass = (password || authPromptPassword.value || CONSTANTS.DEFAULT_PASSWORD).padEnd(8, ' ')
            deviceStore.currentDevice.sessionPassword = paddedPass
            
            sessionPasswords.set(portId, paddedPass)
            authRequiredPorts.delete(portId)
            await SetStoredPassword(portId, paddedPass)
        }
        
        showPasswordModal.value = false
        
        let paddedPass = (password || authPromptPassword.value || CONSTANTS.DEFAULT_PASSWORD).padEnd(8, ' ')

        if (pendingAction) {
            const actionToRun = pendingAction
            pendingAction = null
            try {
                await actionToRun()
            } catch(e) {
                console.error("Pending action failed", e)
            }
        }
        
        // After successfully establishing authentication, proactively refresh the device name
        // so that the UI reflects the true name from the device instead of the fallback
        if (deviceStore.currentDevice && deviceStore.currentDevice.portId) {
            try {
                // Wait briefly to ensure previous commands (like SP) have fully cleared the serial port buffer
                await new Promise(r => setTimeout(r, 200))
                
                const resp = await GetDeviceName(deviceStore.currentDevice.portId, paddedPass)
                let normalizedResp = String(resp || '').replace(/[^\x20-\x7E]/g, '')
                
                if (normalizedResp.startsWith('B')) {
                    let payload = normalizedResp.slice(1)
                    if (payload.startsWith('+')) {
                        payload = payload.slice(1)
                    }
                    const finalName = payload.trimEnd()
                    
                    deviceStore.currentDevice.deviceName = finalName
                    deviceStore.currentDevice.displayName = finalName 
                        ? `${finalName} (${deviceStore.currentDevice.portId})` 
                        : `C2G USB Hub Manager (${deviceStore.currentDevice.portId})`
                    
                    const matchedDevice = deviceStore.devices.find(d => d.portId === deviceStore.currentDevice.portId)
                    if (matchedDevice) {
                        matchedDevice.deviceName = deviceStore.currentDevice.deviceName
                        matchedDevice.displayName = deviceStore.currentDevice.displayName
                    }
                    logStore.addLog('System', `Device name updated to "${finalName}"`, deviceStore.currentDevice.portId)
                } else {
                    console.warn(`Failed to auto-refresh name, device responded: ${resp}`)
                }
            } catch (e) {
                console.error("Failed to proactively fetch device name after auth", e)
            }
        }
    }

    const handleSubmitSetPassword = async ({ old, new: newPass, confirm }) => {
        const currentPass = getPreferredPassword(deviceStore.currentDevice?.portId)
        const oldPass = old || currentPass
        const newPadded = (newPass || '').padEnd(8, ' ')

        // Validate
        if (newPass !== confirm) {
            uiStore.showAlert("New password and confirmation do not match.", "Validation Error")
            return
        }
        
        if (newPass.length > 8) {
            uiStore.showAlert("Password must be 8 characters or less.", "Validation Error")
            return
        }
        
        try {
            const resp = await ChangePassword(deviceStore.currentDevice.portId, oldPass, newPass)
            logStore.addLog('User Action', `The password has been set successfully`, deviceStore.currentDevice.portId)
            
            if (resp && (resp.includes('OK') || !resp.includes('E01'))) {
                 deviceStore.currentDevice.sessionPassword = newPadded
                 sessionPasswords.set(deviceStore.currentDevice.portId, newPadded)
                 authRequiredPorts.delete(deviceStore.currentDevice.portId)
                 await SetStoredPassword(deviceStore.currentDevice.portId, newPadded)
                 
                 showSetPasswordModal.value = false
                 uiStore.showAlert("Password updated successfully.", "Success")
            } else {
                 uiStore.showAlert("Failed to update password. Please check the old password.", "Error")
            }
        } catch(e) {
            logStore.addLog('Error', 'Set Password failed: ' + e, deviceStore.currentDevice.portId)
            uiStore.showAlert('Set Password failed: ' + e, "Error")
        }
    }

    const cancelPassword = () => {
        showPasswordModal.value = false
        pendingAction = null
    }

    const checkDeviceAuth = (device) => {
        const portId = device.portId
        device.sessionPassword = authRequiredPorts.has(portId) ? '' : getPreferredPassword(portId)
    }

    const clearSessionPassword = async (portId) => {
        sessionPasswords.delete(portId)
        authRequiredPorts.delete(portId)
        if (deviceStore.currentDevice && deviceStore.currentDevice.portId === portId) {
            deviceStore.currentDevice.sessionPassword = CONSTANTS.DEFAULT_PASSWORD
        }
        await SetStoredPassword(portId, "")
    }

    return {
        showPasswordModal,
        authPromptPassword,
        showSetPasswordModal,
        setPassOld,
        executeWithAuth,
        handlePasswordSubmit,
        handleSubmitSetPassword,
        cancelPassword,
        checkDeviceAuth,
        clearSessionPassword,
        markPasswordRejected
    }
})
