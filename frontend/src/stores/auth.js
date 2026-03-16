import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { GetStoredPassword, SetStoredPassword, SendCommand } from '../../wailsjs/go/main/App'
import { useDeviceStore } from './devices'
import { useLogStore } from './logs'
import { useUIStore } from './ui'

export const useAuthStore = defineStore('auth', () => {
    const deviceStore = useDeviceStore()
    const logStore = useLogStore()
    const uiStore = useUIStore()

    const showPasswordModal = ref(false)
    const showSetPasswordModal = ref(false)
    const setPassOld = ref('')
    const authenticatedPorts = reactive(new Set())
    let pendingAction = null

    const executeWithAuth = async (action, alertFn, forceAuth = false) => {
        const alertFunc = alertFn || uiStore.showAlert
        if (!deviceStore.currentDevice || !deviceStore.currentDevice.portId) {
            if(alertFunc) alertFunc('Please connect a device first.', 'Connection Required')
            return
        }
        const portId = deviceStore.currentDevice.portId
        
        // If forceAuth is true, ignore current session and prompt
        if (!forceAuth) {
            // If we have a session password or authenticated mark FOR THIS SPECIFIC PORT, just proceed
            if (deviceStore.currentDevice.sessionPassword && authenticatedPorts.has(portId)) {
                 action()
                 return
            }
        
            // Try to get password from backend storage first before falling back
            const storedPass = await GetStoredPassword(portId)
            if (storedPass) {
                deviceStore.currentDevice.sessionPassword = storedPass
                if (!authenticatedPorts.has(portId)) authenticatedPorts.add(portId)
                action()
                return
            }
        }
    
        // Force user to input password
        pendingAction = action
        showPasswordModal.value = true
    }

    const handlePasswordSubmit = async (password) => {
        if (!password) return
        
        if (deviceStore.currentDevice) {
            // Save locally for this session
            deviceStore.currentDevice.sessionPassword = password.padEnd(8, ' ')
            authenticatedPorts.add(deviceStore.currentDevice.portId)
            
            // Persist to disk so it survives restart
            await SetStoredPassword(deviceStore.currentDevice.portId, deviceStore.currentDevice.sessionPassword)
        }
        
        showPasswordModal.value = false
        
        if (pendingAction) {
            pendingAction()
            pendingAction = null
        }
    }

    const handleSubmitSetPassword = async ({ old, new: newPass, confirm }) => {
        // Validate
        if (newPass !== confirm) {
            uiStore.showAlert("New password and confirmation do not match.", "Validation Error")
            return
        }
        
        if (newPass.length > 8) {
            uiStore.showAlert("Password must be 8 characters or less.", "Validation Error")
            return
        }
        
        // Command format: CP{old_pass_padded}{new_pass_padded}
        const oldPadded = old.padEnd(8, ' ').substring(0, 8)
        const newPadded = newPass.padEnd(8, ' ').substring(0, 8)
        
        const cmd = `CP${oldPadded}${newPadded}`
        
        try {
            const resp = await SendCommand(deviceStore.currentDevice.portId, cmd)
            logStore.addLog('User Action', `The password has been set successfully`, deviceStore.currentDevice.portId)
            
            // If successful
            if (resp && (resp.includes('OK') || !resp.includes('E01'))) {
                 deviceStore.currentDevice.sessionPassword = newPadded
                 // Persist to disk
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

    const checkDeviceAuth = async (device) => {
        const portId = device.portId
        const storedPass = await GetStoredPassword(portId)
        if (storedPass) {
            device.sessionPassword = storedPass
            if (!authenticatedPorts.has(portId)) authenticatedPorts.add(portId)
        } else {
            device.sessionPassword = null
            authenticatedPorts.delete(portId)
        }
    }

    return {
        showPasswordModal,
        showSetPasswordModal,
        setPassOld,
        authenticatedPorts,
        executeWithAuth,
        handlePasswordSubmit,
        handleSubmitSetPassword,
        cancelPassword,
        checkDeviceAuth
    }
})
