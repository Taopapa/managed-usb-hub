<script setup>
import { defineProps, defineEmits, ref, watch, computed } from 'vue'
import { GetDeviceUID, SetDeviceUID } from '../../wailsjs/go/main/App'
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
const deviceUid = ref('')
const loading = ref(false)
const pending = ref(false)

watch(() => props.show, (visible) => {
  if (visible) {
    deviceUid.value = props.device?.deviceUid || 'C2G 7-port Managed USB HUB'
    executeWithAuth(async () => {
      await loadDeviceUid()
    })
  }
})

watch(() => props.device?.portId, (portId) => {
  if (props.show && portId) {
    deviceUid.value = props.device?.deviceUid || 'C2G 7-port Managed USB HUB'
  }
})

const charCount = computed(() => Array.from(deviceUid.value || '').length)

const normalizeInputDeviceUid = (value) => Array.from(String(value || '')).slice(0, 20).join('')

const formatDeviceUidForCommand = (value) => {
  const normalized = normalizeInputDeviceUid(value)
  return normalized.padEnd(20, ' ')
}

const normalizeDeviceUidPayload = (value) => {
  let val = String(value || '')
  // Replace invalid characters, but keep spaces if they are in the middle of the string
  val = val.replace(/[^\x20-\x7E]/g, '')
  // Then strictly trim the trailing spaces to get the real name
  return val.trimEnd()
}

const parseDeviceUidResponse = (response, expectedPrefix) => {
  let normalizedResp = String(response || '')
  normalizedResp = normalizedResp.replace(/[^\x20-\x7E]/g, '')

  if (!normalizedResp.startsWith(expectedPrefix)) {
    return null
  }

  const payload = normalizedResp.slice(expectedPrefix.length)
  // Need to handle both + and without + responses
  if (payload.startsWith('+')) {
    return normalizeDeviceUidPayload(payload.slice(1))
  }
  return normalizeDeviceUidPayload(payload)
}

const emitUpdatedName = (name) => {
  emit('updated', name || '')
}

const loadDeviceUid = async (attempt = 0) => {
  if (!props.device?.portId) return

  const deviceId = props.device.portId
  const password = props.device.sessionPassword || CONSTANTS.DEFAULT_PASSWORD

  loading.value = true
  try {
    const resp = await GetDeviceUID(deviceId, password)
    const normalizedResp = String(resp || '').trim()

    if (normalizedResp.includes('E01')) {
      await markPasswordRejected(deviceId)

      if (attempt >= CONSTANTS.MAX_AUTH_RETRIES) {
        logStore.addLog('Error', 'Device UID authentication failed during read', deviceId)
        uiStore.showAlert('Device authentication failed. Cannot read the device UID.', 'Error')
        emit('close')
        return
      }

      executeWithAuth(async () => {
        await loadDeviceUid(attempt + 1)
      }, null, true)
      return
    }

    const loadedName = parseDeviceUidResponse(resp, 'I')
    if (loadedName === null) {
      logStore.addLog('Error', `Unexpected response for GetDeviceUID: ${normalizedResp || 'empty response'}`, deviceId)
      uiStore.showAlert(`Failed to read the device UID. Device responded: ${normalizedResp || 'empty response'}`, 'Error')
      emit('close')
      return
    }

    deviceUid.value = loadedName
    emitUpdatedName(loadedName)
  } catch (e) {
    logStore.addLog('Error', 'Device UID read failed: ' + e, deviceId)
    uiStore.showAlert('Device UID read failed: ' + e, 'Error')
    emit('close')
  } finally {
    loading.value = false
  }
}

const close = () => {
  if (pending.value || loading.value) return
  emit('close')
}

const handleDeviceUidInput = (event) => {
  deviceUid.value = normalizeInputDeviceUid(event.target.value)
}

const persistDeviceUid = async (name, attempt = 0) => {
  if (!props.device?.portId) return

  const normalizedName = normalizeInputDeviceUid(name)
  const commandName = formatDeviceUidForCommand(normalizedName)
  const deviceId = props.device.portId
  const password = props.device.sessionPassword || CONSTANTS.DEFAULT_PASSWORD

  pending.value = true
  try {
    const resp = await SetDeviceUID(deviceId, password, commandName)
    const normalizedResp = String(resp || '').trim()

    if (normalizedResp.includes('E01')) {
      await markPasswordRejected(deviceId)

      if (attempt >= CONSTANTS.MAX_AUTH_RETRIES) {
        logStore.addLog('Error', 'Device UID authentication failed during save', deviceId)
        uiStore.showAlert('Device authentication failed. Cannot save the device UID.', 'Error')
        return
      }

      executeWithAuth(async () => {
        await persistDeviceUid(normalizedName, attempt + 1)
      }, null, true)
      return
    }

    const savedName = parseDeviceUidResponse(resp, 'G')
    if (savedName === null) {
      logStore.addLog('Error', `Unexpected response for SetDeviceUID: ${normalizedResp || 'empty response'}`, deviceId)
      uiStore.showAlert(`Failed to save the device UID. Device responded: ${normalizedResp || 'empty response'}`, 'Error')
      return
    }

    // Rely on the response from the device, not what we think we sent
    const finalName = savedName || normalizedName
    deviceUid.value = finalName
    emitUpdatedName(finalName)
    logStore.addLog('User Action', `Device UID updated to "${finalName}"`, deviceId)
    emit('close')
  } catch (e) {
    logStore.addLog('Error', 'Device UID save failed: ' + e, deviceId)
    uiStore.showAlert('Device UID save failed: ' + e, 'Error')
  } finally {
    pending.value = false
  }
}

const save = () => {
  if (charCount.value === 0) {
    uiStore.showAlert('Device UID cannot be empty.', 'Validation Error')
    return
  }
  if (charCount.value > 20) {
    uiStore.showAlert('Device UID must be 20 characters or less.', 'Validation Error')
    return
  }

  executeWithAuth(async () => {
    await persistDeviceUid(deviceUid.value)
  })
}
</script>

<template>
  <div v-if="show" class="modal-overlay">
    <div class="modal">
      <div class="modal-header">
        <span>Device UID</span>
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
          <label for="device-uid-input">Device UID</label>
          <input
            id="device-uid-input"
            :value="deviceUid"
            type="text"
            :disabled="pending || loading"
            autocomplete="off"
            @input="handleDeviceUidInput"
          >
          <div class="field-hint">{{ charCount }}/20 characters</div>
        </div>
        <div v-if="loading" class="status-text">Loading current device UID...</div>
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
