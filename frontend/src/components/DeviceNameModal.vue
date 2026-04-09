<script setup>
import { defineProps, defineEmits, ref, watch, computed } from 'vue'
import { GetDeviceName, SetDeviceName } from '../../wailsjs/go/main/App'
import { useAuthStore } from '../stores/auth'
import { useLogStore } from '../stores/logs'
import { useUIStore } from '../stores/ui'
import { CONSTANTS } from '../config/constants'

const props = defineProps({
  show: Boolean,
  device: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close', 'updated'])
const authStore = useAuthStore()
const logStore = useLogStore()
const uiStore = useUIStore()

const { executeWithAuth, markPasswordRejected } = authStore
const deviceName = ref('')
const loading = ref(false)
const pending = ref(false)

watch(() => props.show, (visible) => {
  if (visible) {
    deviceName.value = props.device?.deviceName || ''
    executeWithAuth(async () => {
      await loadDeviceName()
    })
  }
})

watch(() => props.device?.portId, (portId) => {
  if (props.show && portId) {
    deviceName.value = props.device?.deviceName || ''
  }
})

const charCount = computed(() => Array.from(deviceName.value || '').length)

const normalizeInputDeviceName = (value) => Array.from(String(value || '')).slice(0, 8).join('')

const formatDeviceNameForCommand = (value) => {
  const normalized = normalizeInputDeviceName(value)
  return normalized.padEnd(8, ' ')
}

const normalizeDeviceNamePayload = (value) => {
  let val = String(value || '')
  // Replace invalid characters, but keep spaces if they are in the middle of the string
  val = val.replace(/[^\x20-\x7E]/g, '')
  // Then strictly trim the trailing spaces to get the real name
  return val.trimEnd()
}

const parseDeviceNameResponse = (response, expectedPrefix) => {
  let normalizedResp = String(response || '')
  normalizedResp = normalizedResp.replace(/[^\x20-\x7E]/g, '')

  if (!normalizedResp.startsWith(expectedPrefix)) {
    return null
  }

  const payload = normalizedResp.slice(expectedPrefix.length)
  // Need to handle both + and without + responses
  if (payload.startsWith('+')) {
    return normalizeDeviceNamePayload(payload.slice(1))
  }
  return normalizeDeviceNamePayload(payload)
}

const emitUpdatedName = (name) => {
  emit('updated', name || '')
}

const loadDeviceName = async (attempt = 0) => {
  if (!props.device?.portId) return

  const deviceId = props.device.portId
  const password = props.device.sessionPassword || CONSTANTS.DEFAULT_PASSWORD

  loading.value = true
  try {
    const resp = await GetDeviceName(deviceId, password)
    const normalizedResp = String(resp || '').trim()

    if (normalizedResp.includes('E01')) {
      await markPasswordRejected(deviceId)

      if (attempt >= CONSTANTS.MAX_AUTH_RETRIES) {
        logStore.addLog('Error', 'Device name authentication failed during read', deviceId)
        uiStore.showAlert('Device authentication failed. Cannot read the device name.', 'Error')
        emit('close')
        return
      }

      executeWithAuth(async () => {
        await loadDeviceName(attempt + 1)
      }, null, true)
      return
    }

    const loadedName = parseDeviceNameResponse(resp, 'B')
    if (loadedName === null) {
      logStore.addLog('Error', `Unexpected response for GetDeviceName: ${normalizedResp || 'empty response'}`, deviceId)
      uiStore.showAlert(`Failed to read the device name. Device responded: ${normalizedResp || 'empty response'}`, 'Error')
      emit('close')
      return
    }

    deviceName.value = loadedName
    emitUpdatedName(loadedName)
  } catch (e) {
    logStore.addLog('Error', 'Device name read failed: ' + e, deviceId)
    uiStore.showAlert('Device name read failed: ' + e, 'Error')
    emit('close')
  } finally {
    loading.value = false
  }
}

const close = () => {
  if (pending.value || loading.value) return
  emit('close')
}

const handleDeviceNameInput = (event) => {
  deviceName.value = normalizeInputDeviceName(event.target.value)
}

const persistDeviceName = async (name, attempt = 0) => {
  if (!props.device?.portId) return

  const normalizedName = normalizeInputDeviceName(name)
  const commandName = formatDeviceNameForCommand(normalizedName)
  const deviceId = props.device.portId
  const password = props.device.sessionPassword || CONSTANTS.DEFAULT_PASSWORD

  pending.value = true
  try {
    const resp = await SetDeviceName(deviceId, password, commandName)
    const normalizedResp = String(resp || '').trim()

    if (normalizedResp.includes('E01')) {
      await markPasswordRejected(deviceId)

      if (attempt >= CONSTANTS.MAX_AUTH_RETRIES) {
        logStore.addLog('Error', 'Device name authentication failed during save', deviceId)
        uiStore.showAlert('Device authentication failed. Cannot save the device name.', 'Error')
        return
      }

      executeWithAuth(async () => {
        await persistDeviceName(normalizedName, attempt + 1)
      }, null, true)
      return
    }

    const savedName = parseDeviceNameResponse(resp, 'G')
    if (savedName === null) {
      logStore.addLog('Error', `Unexpected response for SetDeviceName: ${normalizedResp || 'empty response'}`, deviceId)
      uiStore.showAlert(`Failed to save the device name. Device responded: ${normalizedResp || 'empty response'}`, 'Error')
      return
    }

    // Rely on the response from the device, not what we think we sent
    const finalName = savedName || normalizedName
    deviceName.value = finalName
    emitUpdatedName(finalName)
    logStore.addLog('User Action', `Device name updated to "${finalName}"`, deviceId)
    emit('close')
  } catch (e) {
    logStore.addLog('Error', 'Device name save failed: ' + e, deviceId)
    uiStore.showAlert('Device name save failed: ' + e, 'Error')
  } finally {
    pending.value = false
  }
}

const save = () => {
  if (charCount.value > 8) {
    uiStore.showAlert('Device name must be 8 characters or less.', 'Validation Error')
    return
  }

  executeWithAuth(async () => {
    await persistDeviceName(deviceName.value)
  })
}
</script>

<template>
  <div v-if="show" class="modal-overlay">
    <div class="modal">
      <div class="modal-header">
        <span>Device Name</span>
        <button
          class="header-close"
          type="button"
          :disabled="pending || loading"
          aria-label="Close"
          @click="close"
        >&times;</button>
      </div>
      <div class="modal-body form-body">
        <div class="form-group">
          <label for="device-name-input">Device Name</label>
          <input
            id="device-name-input"
            :value="deviceName"
            type="text"
            :disabled="pending || loading"
            autocomplete="off"
            @input="handleDeviceNameInput"
          >
          <div class="field-hint">{{ charCount }}/8 characters</div>
        </div>
        <div v-if="loading" class="status-text">Loading current device name...</div>
      </div>
      <div class="modal-footer">
        <button @click="save" :disabled="pending || loading">Save</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9998;
}

.modal {
  background-color: #fff;
  border: 1px solid #999;
  box-shadow: 2px 2px 10px rgba(0, 0, 0, 0.2);
  width: 340px;
  display: flex;
  flex-direction: column;
}

.modal-header {
  background-color: #0078d7;
  color: #fff;
  padding: 5px 10px;
  font-weight: bold;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.modal-body {
  padding: 15px;
}

.form-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-group label {
  font-size: 12px;
  font-weight: bold;
}

.field-hint,
.status-text {
  color: #666;
  font-size: 12px;
}

.modal-footer {
  padding: 10px;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  background-color: #f0f0f0;
}

input {
  border: 1px solid #ccc;
  padding: 4px 6px;
}

button {
  cursor: pointer;
  background-color: #e1e1e1;
  border: 1px solid #adadad;
  padding: 2px 10px;
}

button:hover:not(:disabled) {
  background-color: #e5f1fb;
  border-color: #0078d7;
}

button:disabled {
  color: #838383;
  background-color: #f0f0f0;
  border-color: #d0d0d0;
}

.header-close {
  min-width: auto;
  padding: 0 6px;
  border: 0;
  background: transparent;
  color: #fff;
  font-size: 18px;
  line-height: 1;
}

.header-close:hover:not(:disabled) {
  background-color: rgba(255, 255, 255, 0.2);
  border-color: transparent;
}
</style>
