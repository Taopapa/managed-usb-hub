import { ExportLogs, OpenSystemTerminal, QuitApp, ChangePassword } from '../../wailsjs/go/main/App'

const APP_VERSION = typeof __APP_VERSION__ !== 'undefined' ? __APP_VERSION__ : 'dev'

export const useAppActions = ({
    currentDevice,
    devices,
    logs,
    showAlert,
    closeMenu,
    autoSearchAction,
    selectDeviceAction,
    checkDeviceAuth,
    loadHistoryLogs,
    authStore
}) => {
    const autoSearch = async (targetDevId) => {
        const targetId = typeof targetDevId === 'string' ? targetDevId : null

        const foundDev = await autoSearchAction(targetId)
        if (foundDev) {
            selectDevice(foundDev)
        } else if (devices.value.length > 0) {
            selectDevice(devices.value[0])
        }
    }

    const selectDevice = async (device) => {
        await selectDeviceAction(device)
        checkDeviceAuth(device)
        loadHistoryLogs(device.portId)
    }

    const handleDeviceSelect = (portId) => {
        const device = devices.value.find(d => d.portId === portId)
        if (device) {
            selectDevice(device)
        }
    }

    const quitApp = () => QuitApp()

    const refreshHub = async () => {
        if (currentDevice.value) {
            const devId = currentDevice.value.portId
            const foundDev = await autoSearchAction(devId)
            if (foundDev) {
                await selectDeviceAction(foundDev)
            }
        }
        closeMenu()
    }

    const runCli = async () => {
        try {
            await OpenSystemTerminal()
        } catch (e) {
            showAlert('Failed to open terminal: ' + e, 'Error')
        }
        closeMenu()
    }

    const exportLogs = async () => {
        try {
            const csvContent = logs.value.map(e => `${e.time},${e.event},${e.deviceID},${e.details}`).join('\n')
            let deviceName = currentDevice.value ? currentDevice.value.portId : 'System'
            deviceName = deviceName.replace(/\//g, '_').replace(/\\/g, '_')
            const fileName = `hub_logs_${deviceName}_${new Date().toISOString().split('T')[0]}.csv`

            await ExportLogs(csvContent, fileName)
        } catch (e) {
            showAlert('Failed to export logs: ' + e, 'Error')
        }
        closeMenu()
    }

    const showAbout = () => {
        showAlert(`C2G USB Hub Manager v${APP_VERSION}\n\nCopyright © 2024-2026 Legrand AV Inc.`, 'About')
        closeMenu()
    }

    const updateSelectedDeviceName = (name) => {
        if (!currentDevice.value) return

        const deviceName = name || ''
        const fallbackDisplayName = `C2G 7-port Managed USB HUB (${currentDevice.value.portId})`

        currentDevice.value.deviceName = deviceName
        currentDevice.value.displayName = deviceName ? `${deviceName} (${currentDevice.value.portId})` : fallbackDisplayName

        const matchedDevice = devices.value.find(d => d.portId === currentDevice.value.portId)
        if (matchedDevice) {
            matchedDevice.deviceName = currentDevice.value.deviceName
            matchedDevice.displayName = currentDevice.value.displayName
        }
    }

    const updateSelectedDeviceUid = (uid) => {
        if (!currentDevice.value) return
        
        currentDevice.value.deviceUid = uid || ''
        
        const matchedDevice = devices.value.find(d => d.portId === currentDevice.value.portId)
        if (matchedDevice) {
            matchedDevice.deviceUid = currentDevice.value.deviceUid
        }
    }

    const clearPassword = () => {
        if (!currentDevice.value) return
        const portId = currentDevice.value.portId
        
        const confirmClear = window.confirm("Are you sure you want to clear the saved password? You will be prompted to enter the password on your next action.")
        if (!confirmClear) return

        // Force the app to require password on the next action by marking it rejected
        authStore.markPasswordRejected(portId)
        
        logStore.addLog('User Action', `Cleared saved session password`, portId)
        
        closeMenu()
    }

    return {
        autoSearch,
        selectDevice,
        handleDeviceSelect,
        quitApp,
        refreshHub,
        runCli,
        exportLogs,
        showAbout,
        updateSelectedDeviceName,
        updateSelectedDeviceUid,
        clearPassword
    }
}
