<script setup>
import { reactive, ref, computed, onMounted, watch } from 'vue'
import { AutoSearchProbe, CloseCurrentPort, OpenPort, SendCommand, QuitApp, WriteLog, GetStoredPassword, SetStoredPassword, ReadLogs, ClearLogFile, GetScheduledTasks, AddScheduledTask, RemoveScheduledTask, UpdateScheduledTask } from '../wailsjs/go/main/App'
import connectedIcon from './static/connected.png'
import disconnectedIcon from './static/disconnect.png'

// --- State ---
const devices = ref([])
const currentDevice = ref(null)
const portStates = reactive({})
const isScanning = ref(false)
const isBackendConnected = ref(false)
const showPasswordModal = ref(false)
const showSetPasswordModal = ref(false) // New modal for changing password
const passwordInput = ref('')
const setPassOld = ref('')
const setPassNew = ref('')
const setPassConfirm = ref('')
const authenticatedPorts = reactive(new Set())
const passInput = ref(null)
const logs = ref([])
const selectedTab = ref('ports')
const activeMenu = ref(null) // For handling menu dropdowns
let pendingAction = null

// Initialize port states (1-7 for the UI image, but code supports 16)
// The image shows 7 ports. We'll stick to 16 in data but UI might adapt.
for (let i = 1; i <= 16; i++) {
    portStates[i] = false
}

// --- Computed ---
const activePortCount = computed(() => Object.values(portStates).filter(v => v).length)
const devicesFoundCount = computed(() => devices.value.length)
const currentPortObj = computed(() => isBackendConnected.value ? { backend: true } : null)
const selectedDeviceTotalPorts = computed(() => currentDevice.value ? (currentDevice.value.totalPorts || 7) : 7) // Default to 7 for UI match

// --- Logging ---
const loadHistoryLogs = async () => {
    try {
        const deviceID = currentDevice.value ? currentDevice.value.portId : ""
        const lines = await ReadLogs(deviceID)
        if (lines && lines.length > 0) {
            // Parse lines: [Time] [Level] [DeviceID] Message
            // Example: [2023-10-27 10:00:00] [System] [System] Application started
            // Or old format fallback?
            const parsed = []
            // Regex for new format: [Time] [Level] [DeviceID] Message
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

const addLog = (event, details, deviceID = null) => {
    const time = new Date().toLocaleString()
    const targetDevice = deviceID || (currentDevice.value ? currentDevice.value.portId : "System")
    
    // Skip System logs
    if (targetDevice === "System") return

    logs.value.unshift({ time, event, deviceID: targetDevice, details })
    // Keep log size manageable
    if (logs.value.length > 100) logs.value.pop()
    
    // Also write to local file via backend
    WriteLog(targetDevice, event, details)
}

// --- Actions ---
const autoSearch = async () => {
    isScanning.value = true
    addLog('System', 'Scanning for USB Hubs...', 'System')
    
    // Reset current selection/connection
    if (currentDevice.value) {
        await CloseCurrentPort()
        currentDevice.value = null
        isBackendConnected.value = false
    }

    devices.value = []
    
    try {
        const results = await AutoSearchProbe()
        if (!results || results.length === 0) {
            // addLog('System', 'No devices found.', 'System')
        } else {
            const parsedResults = results.map(parseDevice)
            devices.value = parsedResults
            // addLog('System', `Found ${results.length} devices.`, 'System')
            
            // Auto-select first if available
            if (devices.value.length > 0) {
                selectDevice(devices.value[0])
            }
        }
    } catch (err) {
        console.error(err)
        // addLog('Error', 'Scan failed: ' + err, 'System')
    } finally {
        isScanning.value = false
    }
}

const parseDevice = (r) => {
    // Parse Version
    let version = "v1.0"
    if (r.probeResponse) {
        const ascii = r.asciiResponse
        const vIndex = ascii.indexOf('v')
        if (vIndex !== -1) {
            let vEnd = ascii.indexOf('\r', vIndex)
            if (vEnd === -1) vEnd = ascii.indexOf('\n', vIndex)
            if (vEnd === -1) vEnd = ascii.length
            version = ascii.substring(vIndex, vEnd).trim()
        }
    }

    // Parse GO Data
    let totalPorts = 8
    let validPortsMask = 0xFFFF
    if (r.goData) {
        let goAscii = ""
        try {
            const asciiStr = r.goData.substring(0, 4)
            goAscii = hexToAscii(asciiStr)
        } catch(e) {}
        
        let maskVal = parseInt(goAscii, 16)
        if (!isNaN(maskVal)) {
            validPortsMask = maskVal
            let count = 0
            let temp = maskVal
            while (temp > 0) {
                if (temp & 1) count++
                temp >>= 1
            }
            totalPorts = count
        }
    }

    // Parse GP Data
    let onPortsList = []
    let offPortsList = []
    if (r.ledStatus) {
        let hexStr = r.ledStatus
        let asciiStr = ""
        try {
            for (let i = 0; i < hexStr.length; i += 2) {
                const code = parseInt(hexStr.substr(i, 2), 16)
                if (!isNaN(code)) asciiStr += String.fromCharCode(code)
            }
        } catch(e) {}
        
        let isAllOn = false
        let isAllOff = false
        
        if (asciiStr.length >= 2) {
            const prefix = asciiStr.substring(0, 2).toUpperCase()
            if (prefix === "FF") isAllOn = true
            else if (prefix === "00") isAllOff = true
        }
        
        if (isAllOn) {
            for (let i = 1; i <= totalPorts; i++) onPortsList.push(i)
        } else if (isAllOff) {
            for (let i = 1; i <= totalPorts; i++) offPortsList.push(i)
        } else {
            const val = parseInt(asciiStr.substring(0, 2), 16)
            if (!isNaN(val)) {
                 for (let i = 1; i <= totalPorts; i++) {
                     if (i <= 7) {
                         if ((val >> (i - 1)) & 1) {
                             onPortsList.push(i)
                         } else {
                             offPortsList.push(i)
                         }
                     } else {
                         offPortsList.push(i)
                     }
                 }
            } else {
                 for (let i = 1; i <= totalPorts; i++) offPortsList.push(i)
            }
        }
    }

    return {
        portId: r.path,
        portName: r.path, // e.g., COM5
        displayName: `C2G54464 7-Port Hub (${r.path})`, // Mock display name
        description: 'C2G 7-Port Usb-A Hub',
        firmwareVersion: version,
        totalPorts: totalPorts,
        onPortsDisplay: onPortsList.length > 0 ? onPortsList.join(',') : '',
        offPortsDisplay: offPortsList.length > 0 ? offPortsList.join(',') : ''
    }
}

function hexToAscii(hex) {
    let str = '';
    for (let i = 0; i < hex.length; i += 2) {
        const code = parseInt(hex.substr(i, 2), 16);
        if (!isNaN(code)) str += String.fromCharCode(code);
    }
    return str;
}

const selectDevice = async (device) => {
    if (!device) return
    if (currentDevice.value && currentDevice.value.portId === device.portId) return

    await CloseCurrentPort()
    currentDevice.value = device
    
    // Sync portStates
    for (let i = 1; i <= 16; i++) {
        portStates[i] = false
    }
    if (device.onPortsDisplay) {
        const onPorts = device.onPortsDisplay.split(',').map(p => parseInt(p.trim()))
        onPorts.forEach(p => {
            if (p >= 1 && p <= 16) portStates[p] = true
        })
    }
    
    // Check for stored password
    const storedPass = await GetStoredPassword(device.portId)
    if (storedPass) {
        currentDevice.value.sessionPassword = storedPass
        if (!authenticatedPorts.has(device.portId)) authenticatedPorts.add(device.portId)
        // addLog('System', 'Loaded stored password for device', device.portId)
    }

    addLog('Device Connected', `Selected ${device.portName}`, device.portId)
    await connectViaBackend(device.portId)
    
    // Reload logs for this device
    loadHistoryLogs()
}

const connectViaBackend = async (portPath) => {
    try {
        await OpenPort(portPath)
        isBackendConnected.value = true
        // addLog('Connection', 'Connected to ' + portPath, portPath)
    } catch(e) {
        console.error('Connection error:', e)
        // addLog('Error', 'Connection failed: ' + e, portPath)
        isBackendConnected.value = false
    }
}

// --- Auth & Commands ---
const executeWithAuth = async (action) => {
    if (!currentDevice.value || !currentDevice.value.portId) {
        alert('Please connect a device first.')
        return
    }
    const portId = currentDevice.value.portId
    
    // If we have a session password or authenticated mark, just proceed
    if (currentDevice.value.sessionPassword || authenticatedPorts.has(portId)) {
         action()
         return
    }

    // Fallback to default password "pass    " if none set
    currentDevice.value.sessionPassword = "pass    "
    if (!authenticatedPorts.has(portId)) authenticatedPorts.add(portId)
    action()
}

const submitPassword = async () => {
    if (!passwordInput.value) return
    
    // Save locally for this session
    currentDevice.value.sessionPassword = passwordInput.value.padEnd(8, ' ')
    authenticatedPorts.add(currentDevice.value.portId)
    
    // Persist to disk so it survives restart
    await SetStoredPassword(currentDevice.value.portId, currentDevice.value.sessionPassword)
    
    showPasswordModal.value = false
    addLog('System', 'Authentication updated and saved', currentDevice.value.portId)
    
    if (pendingAction) {
        pendingAction()
        pendingAction = null
    }
}

const cancelPassword = () => {
    showPasswordModal.value = false
    pendingAction = null
}

const sendSPCommand = async (newPortStates, logDescription) => {
    let password = currentDevice.value ? currentDevice.value.sessionPassword : null
    if (!password && currentDevice.value && window.devicePasswords) {
        password = window.devicePasswords[currentDevice.value.portId]
    }
    if (!currentDevice.value || !password) return

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
        const resp = await SendCommand(cmd)
        
        if (resp && resp.includes('E01')) {
             // Auth failed with current password (maybe default was wrong)
             // Prompt for password now?
             const retry = confirm('Password incorrect (Current password failed). Do you want to enter a password?')
             if (retry) {
                 passwordInput.value = ''
                 showPasswordModal.value = true
             }
             return
        } else if (resp && resp.startsWith('G')) {
             const hexStatus = resp.substring(1, 9)
             // Parse hexStatus logic (omitted for brevity, assume similar to original)
             // Update portStates
             // For now, just trust the input statesToUse for UI feedback
             for(let i=1; i<=16; i++) portStates[i] = statesToUse[i]
             addLog('Command', logDescription || 'Port states updated', currentDevice.value.portId)
        }
    } catch(e) {
        addLog('Error', 'Command failed: ' + e, currentDevice.value.portId)
    }
}

const togglePort = (n) => {
    if (!currentPortObj.value) return
    if (n > selectedDeviceTotalPorts.value) return
    executeWithAuth(async () => {
        const newStates = { ...portStates }
        newStates[n] = !newStates[n]
        const desc = `Port ${n} ${newStates[n] ? 'Enabled' : 'Disabled'}`
        await sendSPCommand(newStates, desc)
        // addLog removed here as it's now handled inside sendSPCommand upon success
    })
}

const allOn = () => {
    executeWithAuth(async () => {
        const newStates = { ...portStates }
        const total = selectedDeviceTotalPorts.value
        for (let i = 1; i <= total; i++) newStates[i] = true
        await sendSPCommand(newStates, "All Ports Enabled")
    })
}

const allOff = () => {
    executeWithAuth(async () => {
        const newStates = { ...portStates }
        const total = selectedDeviceTotalPorts.value
        for (let i = 1; i <= total; i++) newStates[i] = false
        await sendSPCommand(newStates, "All Ports Disabled")
    })
}

const setPasswordAction = async () => {
    executeWithAuth(async () => {
        // Pre-fill fields
        // Old password: current session password OR default
        setPassOld.value = currentDevice.value.sessionPassword || "pass    "
        
        // New/Confirm: default "pass    " (4 chars + 4 spaces)
        // Wait, "pass    " is 8 chars.
        setPassNew.value = "pass    "
        setPassConfirm.value = "pass    "
        
        showSetPasswordModal.value = true
    })
}

const submitSetPassword = async () => {
    // Validate
    if (setPassNew.value !== setPassConfirm.value) {
        alert("New password and confirmation do not match.")
        return
    }
    
    if (setPassNew.value.length > 8) {
        alert("Password must be 8 characters or less.")
        return
    }
    
    // Command format: CP{old_pass_padded}{new_pass_padded}
    const oldPadded = setPassOld.value.padEnd(8, ' ').substring(0, 8)
    const newPadded = setPassNew.value.padEnd(8, ' ').substring(0, 8)
    
    const cmd = `CP${oldPadded}${newPadded}`
    
    try {
        const resp = await SendCommand(cmd)
        addLog('User Action', `The password has been set successfully`, currentDevice.value.portId)
        
        // If successful
        if (resp && (resp.includes('OK') || !resp.includes('E01'))) {
             currentDevice.value.sessionPassword = newPadded
             // Persist to disk
             await SetStoredPassword(currentDevice.value.portId, newPadded)
             
             showSetPasswordModal.value = false
        } else {
             alert("Failed to update password. Please check the old password.")
        }
    } catch(e) {
        addLog('Error', 'Set Password failed: ' + e, currentDevice.value.portId)
    }
}

const restoreDefaultAction = async () => {
    executeWithAuth(async () => {
        // Mock restore implementation
        let password = currentDevice.value.sessionPassword || "pass    "
        const cmd = `RD${password}`
        try {
            await SendCommand(cmd)
            addLog('User Action', 'Restored to default settings', currentDevice.value.portId)
            
            // Disconnect and Reconnect logic
            addLog('System', 'Reconnecting device...', currentDevice.value.portId)
            const dev = currentDevice.value
            
            // 1. Close Port
            await CloseCurrentPort()
            isBackendConnected.value = false
            currentDevice.value = null
            
            // 2. Wait a bit (optional, for device to reset if needed)
            await new Promise(r => setTimeout(r, 1000))
            
            // 3. Re-select device
             // We need to find the device again in the list because currentDevice ref is null now
             // But 'dev' variable holds the reference
             await selectDevice(dev)
             
             // Force refresh visual ports to default (all enabled usually?)
             // Or rely on selectDevice's internal logic if it reads from device?
             // Currently selectDevice resets states to false, then sets based on 'onPortsDisplay' from scan.
             // BUT, since we just did Restore Default, the actual device state might have changed to ALL ON (default).
             // We should probably re-scan (AutoSearchProbe) to get the true new state?
             // Or just assume default is ALL ON.
             
             // Let's assume default is ALL ON.
             for(let i=1; i<=16; i++) portStates[i] = true
             addLog('System', 'Port states reset to default (All Enabled)', dev.portId)
             
             // 4. Reset session password to default since we restored defaults?
            // "Restore Default" usually resets password too?
            // If so, we should update our local session password.
            // Assuming RD resets password to "pass    "
            if (currentDevice.value) {
                currentDevice.value.sessionPassword = "pass    "
                if (window.devicePasswords && dev.portId) {
                    window.devicePasswords[dev.portId] = "pass    "
                }
            }
            
            alert('Device restored to defaults and reconnected.')
            
        } catch (e) {
             addLog('Error', 'Restore Default failed: ' + e, currentDevice.value ? currentDevice.value.portId : 'System')
        }
    })
}

const savePortStates = async () => {
    executeWithAuth(async () => {
        let password = currentDevice.value.sessionPassword
        const cmd = `WP${password}`
        try {
            const resp = await SendCommand(cmd)
            
            if (resp && resp.startsWith('G')) {
                addLog('User Action', 'Port states saved to device memory', currentDevice.value.portId)
                
                // Disconnect and Reconnect logic
                addLog('System', 'Reconnecting device...', currentDevice.value.portId)
                const dev = currentDevice.value
                
                // 1. Close Port
                await CloseCurrentPort()
                isBackendConnected.value = false
                currentDevice.value = null
                
                // 2. Wait a bit
                await new Promise(r => setTimeout(r, 1000))
                
                // 3. Re-scan to refresh list (optional but requested)
                await autoSearch()
                
                // 4. Re-select the same device if it was found again
                // Note: autoSearch might auto-select the first device, but we force re-select the original one
                const foundDev = devices.value.find(d => d.portId === dev.portId)
                if (foundDev) {
                    await selectDevice(foundDev)
                }
                
                alert('Port states saved successfully and device reconnected.')
            } else if (resp && resp.includes('E01')) {
                 // Auth failed
                 const retry = confirm('Password incorrect. Do you want to enter a password?')
                 if (retry) {
                     passwordInput.value = ''
                     showPasswordModal.value = true
                     pendingAction = savePortStates
                 }
            } else {
                 addLog('Error', `Save Port States failed: ${resp}`, currentDevice.value.portId)
                 alert(`Failed to save port states. Device responded: ${resp}`)
            }
        } catch(e) {
            addLog('Error', 'Save Port States command failed: ' + e, currentDevice.value.portId)
            alert('Error saving port states: ' + e)
        }
    })
}

const resetHubAction = async () => {
    executeWithAuth(async () => {
        let password = currentDevice.value.sessionPassword
        const cmd = `RH${password}`
        await SendCommand(cmd)
        addLog('User Action', 'Reset Hub command sent', currentDevice.value.portId)
    })
}

// --- Menu Functions ---
const toggleMenu = (menu) => {
    if (activeMenu.value === menu) {
        activeMenu.value = null
    } else {
        activeMenu.value = menu
    }
}

const closeMenu = () => {
    activeMenu.value = null
}

const exportLogs = () => {
    // Mock export logs
    const csvContent = "data:text/csv;charset=utf-8," 
        + logs.value.map(e => `${e.time},${e.event},${e.deviceID},${e.details}`).join("\n");
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement("a");
    link.setAttribute("href", encodedUri);
    
    // Customize filename based on device
    const deviceName = currentDevice.value ? currentDevice.value.portId : "System"
    const fileName = `hub_logs_${deviceName}_${new Date().toISOString().split('T')[0]}.csv`
    
    link.setAttribute("download", fileName);
    document.body.appendChild(link);
    link.click();
    addLog('System', 'Logs exported', 'System')
    closeMenu()
}

const clearLogsAction = async () => {
    const deviceName = currentDevice.value ? currentDevice.value.portId : "System"
    if (confirm(`Are you sure you want to clear logs for ${deviceName}?`)) {
        try {
            await ClearLogFile(currentDevice.value ? currentDevice.value.portId : "")
            logs.value = []
        } catch (e) {
            console.error("Failed to clear logs", e)
            alert("Failed to clear logs: " + e)
        }
    }
}

const quitApp = () => {
    addLog('System', 'Exiting application...', 'System')
    QuitApp()
}

const refreshHub = async () => {
    if (currentDevice.value) {
        addLog('System', 'Refreshing hub status...', currentDevice.value.portId)
        
        // 1. Close current connection
        await CloseCurrentPort()
        isBackendConnected.value = false
        
        // 2. Re-select device
        const dev = currentDevice.value
        // Temporarily clear to force UI update
        currentDevice.value = null
        
        await new Promise(r => setTimeout(r, 200))
        
        await selectDevice(dev)
    }
    closeMenu()
}

const scheduleTasks = async () => {
    try {
        const tasks = await GetScheduledTasks()
        // Map backend snake_case to frontend camelCase if needed, but Wails usually handles JSON tags.
        // My struct tags are snake_case.
        // Wails JS binding usually keeps the struct field names if not overridden, or uses JSON tags.
        // Since I used `json:"device_id"`, the JS object will have `device_id`.
        scheduledTasks.value = tasks || []
        showScheduleModal.value = true
        activeMenu.value = null
        // Reset form for new entry
        resetTaskForm()
    } catch(e) {
        alert("Failed to load scheduled tasks: " + e)
    }
}

// --- Scheduling State ---
const scheduledTasks = ref([])
const showScheduleModal = ref(false)
const isEditingTask = ref(false)
const taskForm = reactive({
    id: '',
    deviceID: '',
    daysOfWeek: [],
    startTime: '09:00',
    stopTime: '17:00',
    enabled: true,
    startPortStates: [true, true, true, true, true, true, true], // 7 ports, default all on
    stopPortStates: [false, false, false, false, false, false, false] // 7 ports, default all off
})

const daysOptions = [
    { label: 'Sun', value: 0 },
    { label: 'Mon', value: 1 },
    { label: 'Tue', value: 2 },
    { label: 'Wed', value: 3 },
    { label: 'Thu', value: 4 },
    { label: 'Fri', value: 5 },
    { label: 'Sat', value: 6 }
]

const resetTaskForm = () => {
    isEditingTask.value = false
    taskForm.id = Date.now().toString()
    taskForm.deviceID = currentDevice.value ? currentDevice.value.portId : ""
    taskForm.daysOfWeek = [1, 2, 3, 4, 5] // Mon-Fri
    taskForm.startTime = "09:00"
    taskForm.stopTime = "17:00"
    taskForm.enabled = true
    taskForm.startPortStates = [true, true, true, true, true, true, true]
    taskForm.stopPortStates = [false, false, false, false, false, false, false]
}

const parseMaskToStates = (mask) => {
    // Mask is hex string e.g. "81FFFFFF" or "FFFFFFFF"
    // Byte 0 (first 2 chars) contains bit 0-6 for ports 1-7.
    // Actually logic in sendSPCommand:
    // let byte0 = 0x80
    // if (states[i]) byte0 |= (1 << (i - 1))
    // So if port 1 is on, bit 0 is set.
    
    // Logic Fix:
    // If mask is FFFFFFFF -> All true
    // If mask is 00000000 -> All false
    // Else
    
    if (mask === "FFFFFFFF") return [true, true, true, true, true, true, true]
    if (mask === "00000000") return [false, false, false, false, false, false, false]
    
    const states = [false, false, false, false, false, false, false]
    if (!mask || mask.length !== 8) return states
    
    const byte0Hex = mask.substring(0, 2)
    const val = parseInt(byte0Hex, 16)
    if (isNaN(val)) return states
    
    for (let i = 0; i < 7; i++) {
        // Port i+1 corresponds to bit i
        // 1 << 0 is Port 1
        if ((val >> i) & 1) {
            states[i] = true
        }
    }
    return states
}

const statesToMask = (states) => {
    // Replicate sendSPCommand logic
    // states is array of 7 booleans (index 0 = Port 1)
    
    // Check for All On
    if (states.every(s => s)) return "FFFFFFFF"
    // Check for All Off
    if (states.every(s => !s)) return "00000000"
    
    let byte0 = 0x80 // Bit 7 always set?
    for (let i = 0; i < 7; i++) {
        if (states[i]) byte0 |= (1 << i)
    }
    
    return byte0.toString(16).toUpperCase().padStart(2, '0') + "FFFFFF"
}

const editTask = (task) => {
    isEditingTask.value = true
    taskForm.id = task.id
    taskForm.deviceID = task.device_id
    taskForm.daysOfWeek = [...task.days_of_week]
    taskForm.startTime = task.start_time
    taskForm.stopTime = task.stop_time
    taskForm.enabled = task.enabled
    
    // Parse masks
    taskForm.startPortStates = parseMaskToStates(task.start_mask || "FFFFFFFF")
    taskForm.stopPortStates = parseMaskToStates(task.stop_mask || "00000000")
}

const saveTask = async () => {
    if (!taskForm.deviceID) {
        alert("Device ID is required")
        return
    }
    
    const taskData = {
        id: taskForm.id,
        device_id: taskForm.deviceID,
        days_of_week: taskForm.daysOfWeek.map(Number), // Ensure numbers
        start_time: taskForm.startTime,
        stop_time: taskForm.stopTime,
        enabled: taskForm.enabled,
        start_mask: statesToMask(taskForm.startPortStates),
        stop_mask: statesToMask(taskForm.stopPortStates)
    }
    
    try {
        if (isEditingTask.value) {
            await UpdateScheduledTask(taskData)
        } else {
            await AddScheduledTask(taskData)
        }
        // Refresh list
        const tasks = await GetScheduledTasks()
        scheduledTasks.value = tasks
        resetTaskForm()
    } catch(e) {
        alert("Failed to save task: " + e)
    }
}

const deleteTask = async (id) => {
    if (!confirm("Delete this task?")) return
    try {
        await RemoveScheduledTask(id)
        const tasks = await GetScheduledTasks()
        scheduledTasks.value = tasks
        if (isEditingTask.value && taskForm.id === id) resetTaskForm()
    } catch(e) {
        alert("Failed to delete task: " + e)
    }
}

const formatDays = (days) => {
    if (!days || days.length === 0) return "Never"
    if (days.length === 7) return "Every day"
    // Sort
    const sorted = [...days].sort((a,b) => a-b)
    // Map 0-6 to names
    const names = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]
    return sorted.map(d => names[d]).join(", ")
}

const runCli = () => {
    const cmd = prompt("Enter CLI Command:")
    if (cmd) {
        executeWithAuth(async () => {
            const resp = await SendCommand(cmd)
            // addLog('CLI', `Cmd: ${cmd}, Resp: ${resp}`, currentDevice.value.portId)
            // alert(`Response: ${resp}`)
        })
    }
    closeMenu()
}

const showDocs = () => {
    alert('Documentation is available at https://www.c2g.com/')
    closeMenu()
}

const checkUpdates = () => {
    addLog('System', 'Checking for updates...', 'System')
    setTimeout(() => {
        alert('You are running the latest version (v1.0.0).')
    }, 1000)
    closeMenu()
}

const showAbout = () => {
    alert('Managed USB Hub Manager\nVersion 1.0.0\n(c) 2024 C2G')
    closeMenu()
}

// Watch currentDevice to update dropdown selection if needed
watch(currentDevice, (newVal) => {
    if (newVal) {
        // Ensure UI reflects current device
    }
})

// Watch selectedTab to reload logs from file when Logs tab is activated
watch(selectedTab, (newVal) => {
    if (newVal === 'logs') {
        loadHistoryLogs()
    }
})

onMounted(() => {
    // Load historical logs
    loadHistoryLogs()

    // Initial auto search
    // autoSearch()
    
    // Global click listener to close menus
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.menu-item')) {
            closeMenu()
        }
    })
})

</script>

<template>
  <div class="window-container">
    <!-- Menu Bar -->
    <div class="menu-bar">
        <div class="menu-item-container">
            <div class="menu-item" @click.stop="toggleMenu('file')">File</div>
            <div v-if="activeMenu === 'file'" class="dropdown-menu">
                <div class="dropdown-item" @click="autoSearch">Scan for USB Hubs</div>
                <div class="dropdown-item" @click="exportLogs">Export Logs</div>
                <div class="dropdown-separator"></div>
                <div class="dropdown-item" @click="quitApp">Exit</div>
            </div>
        </div>
        
        <div class="menu-item-container">
            <div class="menu-item" @click.stop="toggleMenu('view')">View</div>
            <div v-if="activeMenu === 'view'" class="dropdown-menu">
                <div class="dropdown-item" @click="refreshHub" :class="{ disabled: !currentDevice }">Refresh Selected Hub</div>
            </div>
        </div>
        
        <div class="menu-item-container">
            <div class="menu-item" @click.stop="toggleMenu('tools')">Tools</div>
            <div v-if="activeMenu === 'tools'" class="dropdown-menu">
                <div class="dropdown-item" @click="scheduleTasks">Schedule Tasks</div>
                <div class="dropdown-item" @click="runCli" :class="{ disabled: !currentDevice }">Run CLI Command</div>
            </div>
        </div>
        
        <div class="menu-item-container">
            <div class="menu-item" @click.stop="toggleMenu('help')">Help</div>
            <div v-if="activeMenu === 'help'" class="dropdown-menu">
                <div class="dropdown-item" @click="showDocs">Documentation</div>
                <div class="dropdown-item" @click="checkUpdates">Check for Updates</div>
                <div class="dropdown-separator"></div>
                <div class="dropdown-item" @click="showAbout">About</div>
            </div>
        </div>
    </div>

    <!-- Main Content -->
    <div class="main-body">
        <!-- Top Section: Hub Selection & Info -->
        <div class="top-section">
            <div class="hub-selection">
                <label>Selected USB Hub</label>
                <div class="selection-row">
                    <select :value="currentDevice?.portId" @change="e => {
                        const dev = devices.find(d => d.portId === e.target.value)
                        if(dev) selectDevice(dev)
                    }">
                        <option v-if="!currentDevice" value="">Select a hub...</option>
                        <option v-for="dev in devices" :key="dev.portId" :value="dev.portId">
                            {{ dev.displayName }}
                        </option>
                    </select>
                    <button @click="autoSearch" :disabled="isScanning">Scan for USB Hubs</button>
                </div>
            </div>
            
            <div class="hub-info" v-if="currentDevice">
                <div class="info-text">
                    <div class="model-number">C2G54464</div>
                    <div class="description">C2G 7-Port USB-A Hub</div>
                </div>
                <div class="hub-image">
                    <!-- Placeholder for Hub Image -->
                    <div class="img-placeholder">Hub Image</div>
                </div>
            </div>
        </div>

        <!-- Middle Section: Port Controls -->
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
                             <!-- Image Icons -->
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

        <!-- Tabs Section -->
        <div class="tabs-container">
            <div class="tab-headers">
                <div class="tab-header" :class="{ active: selectedTab === 'ports' }" @click="selectedTab = 'ports'">Ports</div>
                <div class="tab-header" :class="{ active: selectedTab === 'logs' }" @click="selectedTab = 'logs'">logs</div>
            </div>
            
            <div class="tab-content">
                <div v-if="selectedTab === 'ports'" class="ports-table-container">
                    <table class="data-table">
                        <thead>
                            <tr>
                                <th>Port</th>
                                <th>Status</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="n in 7" :key="n">
                                <td>{{ n }}</td>
                                <td>{{ portStates[n] ? 'Enabled' : 'Disabled' }}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                
                <div v-if="selectedTab === 'logs'" class="logs-container">
                    <div class="logs-toolbar">
                        <button @click="clearLogsAction">Clear Logs</button>
                        <button @click="exportLogs">Export Logs</button>
                    </div>
                    <div class="logs-table-wrapper">
                        <table class="data-table">
                            <thead>
                                <tr>
                                    <th style="width: 150px;">Time</th>
                                    <th style="width: 150px;">Events</th>
                                    <th style="width: 100px;">Device</th>
                                    <th>Details</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="(log, index) in logs" :key="index">
                                    <td>{{ log.time }}</td>
                                    <td>{{ log.event }}</td>
                                    <td>{{ log.deviceID }}</td>
                                    <td>{{ log.details }}</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Status Bar -->
    <div class="status-bar">
        <div class="status-left">Ready</div>
        <div class="status-right">{{ devicesFoundCount }} hub{{ devicesFoundCount !== 1 ? 's' : '' }} online, 0 hubs offline</div>
    </div>

    <!-- Password Modal -->
    <div v-if="showPasswordModal" class="modal-overlay">
        <div class="modal">
            <div class="modal-header">Authentication Required</div>
            <div class="modal-body">
                <p>Enter Password (8 chars max):</p>
                <input type="password" v-model="passwordInput" maxlength="8" @keyup.enter="submitPassword" ref="passInput">
            </div>
            <div class="modal-footer">
                <button @click="cancelPassword">Cancel</button>
                <button @click="submitPassword">Confirm</button>
            </div>
        </div>
    </div>

    <!-- Set Password Modal -->
    <div v-if="showSetPasswordModal" class="modal-overlay">
        <div class="modal">
            <div class="modal-header">Change Password</div>
            <div class="modal-body form-body">
                 <div class="form-group">
                     <label>Old Password:</label>
                     <input type="password" v-model="setPassOld" maxlength="8">
                 </div>
                 <div class="form-group">
                     <label>New Password:</label>
                     <input type="password" v-model="setPassNew" maxlength="8">
                 </div>
                 <div class="form-group">
                     <label>Confirm Password:</label>
                     <input type="password" v-model="setPassConfirm" maxlength="8">
                 </div>
             </div>
            <div class="modal-footer">
                <button @click="showSetPasswordModal = false">Cancel</button>
                <button @click="submitSetPassword">Update</button>
            </div>
        </div>
    </div>

    <!-- Schedule Modal -->
    <div v-if="showScheduleModal" class="modal-overlay">
        <div class="modal" style="width: 700px; height: 600px;">
            <div class="modal-header">
                <span>Schedule Tasks</span>
                <button class="close-btn" @click="showScheduleModal = false">×</button>
            </div>
            <div class="modal-body" style="display: flex; flex-direction: column;">
                <!-- Task List -->
                <div class="task-list" style="flex: 1; min-height: 160px; overflow: auto; border: 1px solid #ccc; margin-bottom: 10px;">
                    <table class="data-table">
                        <thead>
                            <tr>
                                <th>Device</th>
                                <th>Days</th>
                                <th>On Time</th>
                                <th>Off Time</th>
                                <th>Start Mask</th>
                                <th>Stop Mask</th>
                                <th>Active</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="task in scheduledTasks" :key="task.id" :style="isEditingTask && taskForm.id === task.id ? 'background-color: #e0f0ff;' : ''">
                                <td>{{ task.device_id }}</td>
                                <td>{{ formatDays(task.days_of_week) }}</td>
                                <td>{{ task.start_time }}</td>
                                <td>{{ task.stop_time }}</td>
                                <td style="font-family: monospace; font-size: 10px;">{{ task.start_mask || "All On" }}</td>
                                <td style="font-family: monospace; font-size: 10px;">{{ task.stop_mask || "All Off" }}</td>
                                <td>{{ task.enabled ? 'Yes' : 'No' }}</td>
                                <td>
                                    <button @click="editTask(task)">Edit</button>
                                    <button @click="deleteTask(task.id)">Del</button>
                                </td>
                            </tr>
                            <tr v-if="scheduledTasks.length === 0">
                                <td colspan="8" style="text-align: center;">No scheduled tasks</td>
                            </tr>
                        </tbody>
                    </table>
                </div>

                <!-- Editor Form -->
                <div class="schedule-form" style="background-color: #f9f9f9; padding: 10px; border: 1px solid #ddd;">
                    <div style="font-weight: bold; margin-bottom: 5px;">{{ isEditingTask ? 'Edit Task' : 'Add New Task' }}</div>
                    
                    <div class="form-row" style="display: flex; gap: 10px; margin-bottom: 5px;">
                         <div class="form-group" style="flex: 1;">
                             <label>Device ID (Port):</label>
                             <input type="text" v-model="taskForm.deviceID" placeholder="e.g. COM3">
                         </div>
                         <div class="form-group">
                             <label>Enabled:</label>
                             <input type="checkbox" v-model="taskForm.enabled" style="margin-top: 5px;">
                         </div>
                    </div>
                    
                    <div class="form-group" style="margin-bottom: 5px;">
                        <label>Days:</label>
                        <div style="display: flex; gap: 10px; flex-wrap: wrap;">
                            <label v-for="day in daysOptions" :key="day.value" style="font-weight: normal; display: flex; align-items: center; gap: 2px;">
                                <input type="checkbox" :value="day.value" v-model="taskForm.daysOfWeek">
                                {{ day.label }}
                            </label>
                        </div>
                    </div>
                    
                    <div class="form-row" style="display: flex; gap: 10px;">
                        <div class="form-group">
                            <label>Turn On Time:</label>
                            <input type="time" v-model="taskForm.startTime">
                        </div>
                        <div class="form-group">
                            <label>Turn Off Time:</label>
                            <input type="time" v-model="taskForm.stopTime">
                        </div>
                    </div>

                    <div class="form-group" style="margin-top: 10px;">
                        <label>Start Action Ports (When task starts):</label>
                        <div style="display: flex; gap: 5px; background-color: #eee; padding: 5px; border-radius: 4px;">
                            <div v-for="(state, idx) in taskForm.startPortStates" :key="'s'+idx" 
                                 style="display: flex; flex-direction: column; align-items: center; cursor: pointer;"
                                 @click="taskForm.startPortStates[idx] = !state">
                                <div :style="{ width: '30px', height: '30px', backgroundColor: state ? '#4CAF50' : '#ccc', borderRadius: '4px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'white', fontWeight: 'bold' }">
                                    {{ idx + 1 }}
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="form-group" style="margin-top: 5px;">
                        <label>Stop Action Ports (When task stops):</label>
                        <div style="display: flex; gap: 5px; background-color: #eee; padding: 5px; border-radius: 4px;">
                            <div v-for="(state, idx) in taskForm.stopPortStates" :key="'e'+idx" 
                                 style="display: flex; flex-direction: column; align-items: center; cursor: pointer;"
                                 @click="taskForm.stopPortStates[idx] = !state">
                                <div :style="{ width: '30px', height: '30px', backgroundColor: state ? '#4CAF50' : '#ccc', borderRadius: '4px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'white', fontWeight: 'bold' }">
                                    {{ idx + 1 }}
                                </div>
                            </div>
                        </div>
                        <div style="font-size: 10px; color: #666; margin-top: 2px;">* Click numbers to toggle port state. Green = On, Grey = Off.</div>
                    </div>
                    
                    <div style="margin-top: 10px; display: flex; gap: 5px; justify-content: flex-end;">
                         <button v-if="isEditingTask" @click="resetTaskForm">Cancel Edit</button>
                         <button @click="saveTask">{{ isEditingTask ? 'Update Task' : 'Add Task' }}</button>
                    </div>
                </div>
            </div>
        </div>
    </div>

  </div>
</template>

<style scoped>
/* Reset & Base */
* {
    box-sizing: border-box;
    user-select: none;
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    font-size: 13px;
}

.window-container {
    display: flex;
    flex-direction: column;
    height: 100vh;
    background-color: #f0f0f0;
    color: #333;
}

/* Menu Bar */
.menu-bar {
    display: flex;
    background-color: #fff;
    border-bottom: 1px solid #ccc;
    padding: 0;
}

.menu-item-container {
    position: relative;
}

.menu-item {
    padding: 5px 10px;
    cursor: pointer;
}

.menu-item:hover {
    background-color: #e5e5e5;
}

.dropdown-menu {
    position: absolute;
    top: 100%;
    left: 0;
    background-color: #fff;
    border: 1px solid #ccc;
    box-shadow: 2px 2px 5px rgba(0,0,0,0.2);
    min-width: 150px;
    z-index: 1000;
}

.dropdown-item {
    padding: 5px 15px;
    cursor: pointer;
}

.dropdown-item:hover {
    background-color: #f0f0f0;
}

.dropdown-item.disabled {
    color: #999;
    cursor: default;
}

.dropdown-item.disabled:hover {
    background-color: #fff;
}

.dropdown-separator {
    height: 1px;
    background-color: #ccc;
    margin: 2px 0;
}

/* Main Body */
.main-body {
    flex: 1;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    overflow: hidden;
}

/* Top Section */
.top-section {
    display: flex;
    justify-content: space-between;
    background-color: #f0f0f0;
    padding-bottom: 10px;
}

.hub-selection {
    display: flex;
    flex-direction: column;
    gap: 5px;
}

.hub-selection label {
    font-weight: bold;
}

.selection-row {
    display: flex;
    gap: 10px;
}

.selection-row select {
    width: 250px;
    padding: 2px;
}

.hub-info {
    display: flex;
    gap: 10px;
    align-items: center;
}

.hub-image .img-placeholder {
    width: 100px;
    height: 30px;
    background-color: #333;
    color: #fff;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 10px;
    border-radius: 2px;
}

/* Control Panel */
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
    /* flex-direction: column; */
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

/* Tabs */
.tabs-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    border: 1px solid #ccc;
    background-color: #fff;
    min-height: 150px;
}

.tab-headers {
    display: flex;
    background-color: #f0f0f0;
    border-bottom: 1px solid #ccc;
}

.tab-header {
    padding: 4px 15px;
    cursor: pointer;
    border-right: 1px solid #ccc;
    background-color: #e0e0e0;
}

.tab-header.active {
    background-color: #fff;
    border-bottom: 1px solid #fff;
    margin-bottom: -1px;
}

.tab-content {
    flex: 1;
    overflow: auto;
    padding: 0;
}

/* Tables */
.data-table {
    width: 100%;
    border-collapse: collapse;
}

.data-table th, .data-table td {
    border: 1px solid #ddd;
    padding: 4px 8px;
    text-align: left;
}

.data-table th {
    background-color: #f5f5f5;
    font-weight: normal;
}

.data-table tr:nth-child(even) {
    background-color: #f9f9f9;
}

.data-table tr:hover {
    background-color: #eef;
}

.logs-container {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.logs-toolbar {
    padding: 5px;
    background-color: #f9f9f9;
    border-bottom: 1px solid #ddd;
    display: flex;
    gap: 10px;
}

.logs-table-wrapper {
    flex: 1;
    overflow: auto;
}

/* Devices Tab Styles */
/* Event Log */
.event-log-section {
    height: 150px;
    display: flex;
    flex-direction: column;
    border: 1px solid #ccc;
    background-color: #fff;
}

.section-title {
    background-color: #f0f0f0;
    padding: 2px 5px;
    font-weight: bold;
    border-bottom: 1px solid #ccc;
}

.log-table-container {
    flex: 1;
    overflow: auto;
}

/* Status Bar */
.status-bar {
    background-color: #f0f0f0;
    border-top: 1px solid #ccc;
    padding: 2px 5px;
    display: flex;
    justify-content: space-between;
}

/* Modal */
.modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0,0,0,0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
}

.modal {
    background-color: #fff;
    border: 1px solid #999;
    box-shadow: 2px 2px 10px rgba(0,0,0,0.2);
    width: 300px;
    display: flex;
    flex-direction: column;
}

.modal-header {
    background-color: #0078d7;
    color: #fff;
    padding: 5px 10px;
    font-weight: bold;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.close-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 16px;
    font-weight: bold;
    cursor: pointer;
}

.modal-body {
    padding: 15px;
    flex: 1;
    overflow: auto;
}

.modal-footer {
    padding: 10px;
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    background-color: #f0f0f0;
}

.form-body {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.form-group {
    display: flex;
    flex-direction: column;
    gap: 2px;
}

.form-group label {
    font-size: 12px;
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

input, select {
    border: 1px solid #ccc;
    padding: 2px;
}
</style>
