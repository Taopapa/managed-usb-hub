<script setup>
import { ref, onMounted, watch, provide } from 'vue'
import AlertModal from './components/AlertModal.vue'
import ConfirmModal from './components/ConfirmModal.vue'
import PasswordModal from './components/PasswordModal.vue'
import SetPasswordModal from './components/SetPasswordModal.vue'
import DocumentationModal from './components/DocumentationModal.vue'
import DeviceNameModal from './components/DeviceNameModal.vue'
import DeviceUidModal from './components/DeviceUidModal.vue'
import VBUSPowerModal from './components/VBUSPowerModal.vue'
import DeviceList from './components/DeviceList.vue'
import DeviceInfo from './components/DeviceInfo.vue'
import PortList from './components/PortList.vue'
import LogList from './components/LogList.vue'
import ControlPanel from './components/ControlPanel.vue'

import { useDeviceStore } from './stores/devices'
import { useLogStore } from './stores/logs'
import { useAuthStore } from './stores/auth'
import { useUIStore } from './stores/ui'
import { useAppActions } from './composables/useAppActions'
import { useAppMenus } from './composables/useAppMenus'
import { useMenuModals } from './composables/useMenuModals'
import { storeToRefs } from 'pinia'
import { EventsOn } from '../wailsjs/runtime'

const deviceStore = useDeviceStore()
const logStore = useLogStore()
const authStore = useAuthStore()
const uiStore = useUIStore()

const { devices, currentDevice, portStates, isScanning, devicesFoundCount, selectedDeviceTotalPorts } = storeToRefs(deviceStore)
const { logs } = storeToRefs(logStore)
const { showPasswordModal, authPromptPassword, showSetPasswordModal, setPassOld } = storeToRefs(authStore)
const { alert: customAlert, confirmState } = storeToRefs(uiStore)

const { autoSearch: autoSearchAction, selectDevice: selectDeviceAction } = deviceStore
const { loadHistoryLogs: loadLogsAction } = logStore
const { checkDeviceAuth, handlePasswordSubmit, handleSubmitSetPassword, cancelPassword } = authStore
const { showAlert, closeAlert, handleConfirmResult } = uiStore

const loadHistoryLogs = async (id) => loadLogsAction(id)

provide('showAlert', showAlert)

// --- Event Listeners ---
onMounted(() => {
    loadHistoryLogs()
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.menu-item')) closeMenu()
    })

    try {
        EventsOn("backend-error", (data) => {
            if (data && data.message) {
                logStore.addLog("System Error", data.message, data.deviceID || "System")
            }
        })
    } catch(e) {
        console.error("Failed to register events", e)
    }
})

// --- Menu Functions ---
const activeMenu = ref(null)
const toggleMenu = (menu) => {
    activeMenu.value = activeMenu.value === menu ? null : menu
}
const closeMenu = () => {
    activeMenu.value = null
}

// --- Scheduling ---
const {
    showDocModal,
    showVBUSPowerModal,
    showDeviceNameModal,
    showDeviceUidModal,
    openDeviceNameModal,
    closeDeviceNameModal,
    openDeviceUidModal,
    closeDeviceUidModal,
    openVBUSPowerModal,
    closeVBUSPowerModal,
    openDocumentationModal,
    closeDocumentationModal
} = useMenuModals({
    currentDevice,
    showAlert,
    closeMenu
})

const {
    autoSearch,
    handleDeviceSelect,
    quitApp,
    refreshHub,
    runCli,
    exportLogs,
    showAbout,
    updateSelectedDeviceName,
    updateSelectedDeviceUid,
    clearPassword
} = useAppActions({
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
})

const menus = useAppMenus({
    currentDevice,
    autoSearch,
    exportLogs,
    quitApp,
    refreshHub,
    openDeviceNameModal,
    openDeviceUidModal,
    openVBUSPowerModal,
    runCli,
    openDocumentationModal,
    showAbout,
    clearPassword
})

// Watchers
const selectedTab = ref('ports')
watch(selectedTab, (newVal) => {
    if (newVal === 'logs') loadHistoryLogs()
})

</script>

<template>
  <div class="window-container">
    <!-- Menu Bar -->
    <div class="menu-bar">
        <div v-for="menu in menus" :key="menu.key" class="menu-item-container">
            <div class="menu-item" @click.stop="toggleMenu(menu.key)">{{ menu.label }}</div>
            <div v-if="activeMenu === menu.key" class="dropdown-menu">
                <template v-for="item in menu.items" :key="item.key">
                    <div v-if="item.separator" class="dropdown-separator"></div>
                    <div
                        v-else
                        class="dropdown-item"
                        :class="{ disabled: item.disabled }"
                        @click="!item.disabled && item.onClick()"
                    >{{ item.label }}</div>
                </template>
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
                        :total-ports="selectedDeviceTotalPorts" 
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
        <div class="status-left">
            <span v-if="currentDevice?.deviceUid">UID: {{ currentDevice.deviceUid }}</span>
            <span v-else></span>
        </div>
        <div class="status-right">{{ devicesFoundCount }} hub{{ devicesFoundCount !== 1 ? 's' : '' }} online</div>
    </div>

    <!-- Password Modal -->
    <PasswordModal 
        :show="showPasswordModal" 
        :initialPassword="authPromptPassword"
        @close="cancelPassword" 
        @submit="handlePasswordSubmit" 
    />

    <!-- Set Password Modal -->
    <SetPasswordModal
        :show="showSetPasswordModal"
        :initialOld="setPassOld"
        @close="showSetPasswordModal = false"
        @submit="handleSubmitSetPassword"
    />

    <!-- Documentation Modal -->
    <DocumentationModal 
        :show="showDocModal" 
        @close="closeDocumentationModal" 
    />

    <!-- Device Name Modal -->
    <DeviceNameModal
        :show="showDeviceNameModal"
        :device="currentDevice"
        @close="closeDeviceNameModal"
        @updated="updateSelectedDeviceName"
    />

    <!-- Device UID Modal -->
    <DeviceUidModal
        :show="showDeviceUidModal"
        :device="currentDevice"
        @close="closeDeviceUidModal"
        @updated="updateSelectedDeviceUid"
    />

    <!-- VBUS Power Modal -->
    <VBUSPowerModal
        :show="showVBUSPowerModal"
        :device="currentDevice"
        @close="closeVBUSPowerModal"
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
        :confirm-label="confirmState.confirmLabel"
        :cancel-label="confirmState.cancelLabel"
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
