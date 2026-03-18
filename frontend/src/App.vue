<script setup>
import { reactive, ref, onMounted, watch, provide } from 'vue'
import { QuitApp, SetStoredPassword, OpenSystemTerminal, ExportLogs } from '../wailsjs/go/main/App'
import AlertModal from './components/AlertModal.vue'
import ConfirmModal from './components/ConfirmModal.vue'
import PasswordModal from './components/PasswordModal.vue'
import SetPasswordModal from './components/SetPasswordModal.vue'
import ScheduleModal from './components/ScheduleModal.vue'
import DocumentationModal from './components/DocumentationModal.vue'
import DeviceList from './components/DeviceList.vue'
import DeviceInfo from './components/DeviceInfo.vue'
import PortList from './components/PortList.vue'
import LogList from './components/LogList.vue'
import ControlPanel from './components/ControlPanel.vue'

import { useDeviceStore } from './stores/devices'
import { useLogStore } from './stores/logs'
import { useAuthStore } from './stores/auth'
import { useUIStore } from './stores/ui'
import { storeToRefs } from 'pinia'
import { EventsOn } from '../wailsjs/runtime'

const deviceStore = useDeviceStore()
const logStore = useLogStore()
const authStore = useAuthStore()
const uiStore = useUIStore()

const { devices, currentDevice, portStates, isScanning, isBackendConnected, devicesFoundCount } = storeToRefs(deviceStore)
const { logs } = storeToRefs(logStore)
const { showPasswordModal, showSetPasswordModal, setPassOld, authenticatedPorts } = storeToRefs(authStore)
const { alert: customAlert, confirmState } = storeToRefs(uiStore)

const { autoSearch: autoSearchAction, selectDevice: selectDeviceAction, disconnect, updatePortStatesFromHex } = deviceStore
const { loadHistoryLogs: loadLogsAction, addLog: addLogAction } = logStore
const { checkDeviceAuth, handlePasswordSubmit, handleSubmitSetPassword, cancelPassword } = authStore
const { showAlert, closeAlert, showConfirm, handleConfirmResult } = uiStore

const addLog = (event, details, deviceID = null) => addLogAction(event, details, deviceID)
const loadHistoryLogs = async (id) => loadLogsAction(id)

provide('showAlert', showAlert)

// --- Event Listeners ---
onMounted(() => {
    loadHistoryLogs()
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.menu-item')) closeMenu()
    })

    // Listen for scheduled task execution events
    try {
        EventsOn("task-executed", (data) => {
             // Convert map to object if needed (Wails might return map as object)
             // data is { deviceID, mask, response, timestamp }
             console.log("Task executed event received:", data)
             
             if (data && data.deviceID) {
                 // 1. Add log entry
                 addLog("Scheduler", `Task executed. Mask: ${data.mask}`, data.deviceID)
                 
                 // 2. Update UI if we are looking at this device
                 if (currentDevice.value && currentDevice.value.portId === data.deviceID) {
                     if (data.response && data.response.startsWith('G')) {
                         const hexStatus = data.response.substring(1) // Remove 'G'
                         console.log("Updating port states from hex:", hexStatus)
                         updatePortStatesFromHex(hexStatus)
                     }
                 }
             }
        })
    } catch (e) {
        console.error("Failed to setup event listener", e)
    }
})

// --- Actions ---
const autoSearch = async (targetDevId) => {
    // If targetDevId is an event object (click event), ignore it
    const targetId = (typeof targetDevId === 'string') ? targetDevId : null
    
    const foundDev = await autoSearchAction(targetId)
    if (foundDev) {
        selectDevice(foundDev)
    } else if (devices.value.length > 0) {
        selectDevice(devices.value[0])
    }
}

const selectDevice = async (device) => {
    await selectDeviceAction(device)
    // Check auth state (syncs session password)
    await checkDeviceAuth(device)
    loadHistoryLogs(device.portId)
}

const handleDeviceSelect = (portId) => {
    const dev = devices.value.find(d => d.portId === portId)
    if(dev) selectDevice(dev)
}

// --- Menu Functions ---
const activeMenu = ref(null)
const toggleMenu = (menu) => {
    activeMenu.value = activeMenu.value === menu ? null : menu
}
const closeMenu = () => {
    activeMenu.value = null
}

const quitApp = () => QuitApp()

const refreshHub = async () => {
    if (currentDevice.value) {
        const dev = currentDevice.value
        // Don't disconnect, just re-select which triggers re-connect logic
        await selectDevice(dev)
    }
    closeMenu()
}

// --- Scheduling ---
const showScheduleModal = ref(false)
const showDocModal = ref(false)

const scheduleTasks = () => {
    showScheduleModal.value = true
    activeMenu.value = null
}

// CLI
const runCli = async () => {
    try {
        await OpenSystemTerminal()
    } catch(e) {
        showAlert("Failed to open terminal: " + e, "Error")
    }
    closeMenu()
}

const clearPasswordAction = async () => {
    if (!currentDevice.value || !currentDevice.value.portId) return
    
    const confirmed = await showConfirm(`Are you sure you want to clear the stored password for ${currentDevice.value.portId}?`, 'Confirm Clear Password')
    
    if (confirmed) {
        try {
            await SetStoredPassword(currentDevice.value.portId, "")
            if (window.devicePasswords && window.devicePasswords[currentDevice.value.portId]) {
                delete window.devicePasswords[currentDevice.value.portId]
            }
            currentDevice.value.sessionPassword = null
            authenticatedPorts.value.delete(currentDevice.value.portId)
            
            showAlert('Password cleared successfully.', "Success")
            
            // Reconnect to force re-auth
            refreshHub()
            
        } catch(e) {
            showAlert('Failed to clear password: ' + e, "Error")
        }
    }
    closeMenu()
}

const exportLogs = async () => {
    try {
        const csvContent = logs.value.map(e => `${e.time},${e.event},${e.deviceID},${e.details}`).join("\n");
        const deviceName = currentDevice.value ? currentDevice.value.portId : "System"
        const fileName = `hub_logs_${deviceName}_${new Date().toISOString().split('T')[0]}.csv`
        
        await ExportLogs(csvContent, fileName)
    } catch (e) {
        showAlert("Failed to export logs: " + e, "Error")
    }
    closeMenu()
}

const showDocs = () => {
    showDocModal.value = true
    closeMenu()
}

const checkUpdates = () => {
    setTimeout(() => showAlert('You are running the latest version (v1.0.0).', "Updates"), 1000)
    closeMenu()
}

const showAbout = () => {
    showAlert('Managed USB Hub Manager\nVersion 1.0.0\n(c) 2024 C2G', "About")
    closeMenu()
}

// Watchers
const selectedTab = ref('ports')
watch(selectedTab, (newVal) => {
    if (newVal === 'logs') loadHistoryLogs()
})

onMounted(() => {
    loadHistoryLogs()
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.menu-item')) closeMenu()
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
                <div class="dropdown-separator"></div>
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
                <div class="dropdown-separator"></div>
                <div class="dropdown-item" @click="runCli">Run CLI Command</div>
                <div class="dropdown-separator"></div>
                <div class="dropdown-item" @click="clearPasswordAction" :class="{ disabled: !currentDevice }">Clear Stored Password</div>
            </div>
        </div>
        
        <div class="menu-item-container">
            <div class="menu-item" @click.stop="toggleMenu('help')">Help</div>
            <div v-if="activeMenu === 'help'" class="dropdown-menu">
                <div class="dropdown-item" @click="showDocs">Documentation</div>
                <!-- <div class="dropdown-item" @click="checkUpdates">Check for Updates</div> -->
                <div class="dropdown-separator"></div>
                <div class="dropdown-item" @click="showAbout">About</div>
            </div>
        </div>
    </div>

    <!-- Main Content -->
    <div class="main-body">
        <!-- Top Section: Hub Selection & Info -->
        <div class="top-section">
            <DeviceList 
                :devices="devices" 
                :current-device-id="currentDevice?.portId" 
                :is-scanning="isScanning" 
                @select="handleDeviceSelect" 
                @scan="autoSearch" 
            />
            
            <DeviceInfo :device="currentDevice" />
        </div>

        <!-- Middle Section: Port Controls -->
        <ControlPanel />

        <!-- Tabs Section -->
        <div class="tabs-container">
            <div class="tab-headers">
                <div class="tab-header" :class="{ active: selectedTab === 'ports' }" @click="selectedTab = 'ports'">Ports</div>
                <div class="tab-header" :class="{ active: selectedTab === 'logs' }" @click="selectedTab = 'logs'">Logs</div>
            </div>
            
            <div class="tab-content">
                <div v-if="selectedTab === 'ports'">
                    <PortList 
                        :port-states="portStates" 
                        :total-ports="7" 
                    />
                </div>
                
                <div v-if="selectedTab === 'logs'">
                    <LogList 
                        :current-device-id="currentDevice?.portId" 
                    />
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
    <PasswordModal 
        :show="showPasswordModal" 
        @close="cancelPassword" 
        @submit="handlePasswordSubmit" 
    />

    <!-- Set Password Modal -->
    <SetPasswordModal 
        :show="showSetPasswordModal" 
        :initial-old="setPassOld" 
        @close="showSetPasswordModal = false" 
        @submit="handleSubmitSetPassword" 
    />

    <!-- Schedule Modal -->
    <ScheduleModal 
        :show="showScheduleModal" 
        :devices="devices"
        @close="showScheduleModal = false" 
    />

    <!-- Documentation Modal -->
    <DocumentationModal 
        :show="showDocModal" 
        @close="showDocModal = false" 
    />

    <!-- Custom Alert Modal -->
    <AlertModal 
        :show="customAlert.show" 
        :title="customAlert.title" 
        :message="customAlert.message" 
        @close="closeAlert"
    />

    <!-- Custom Confirm Modal -->
    <ConfirmModal 
        :show="confirmState.show" 
        :title="confirmState.title" 
        :message="confirmState.message" 
        @result="handleConfirmResult"
    />
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

/* Status Bar */
.status-bar {
    background-color: #f0f0f0;
    border-top: 1px solid #ccc;
    padding: 2px 5px;
    display: flex;
    justify-content: space-between;
}
.top-section{
    display: flex;
    justify-content: space-between;
}
</style>
