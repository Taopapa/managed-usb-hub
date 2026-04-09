<script setup>
import { defineProps, defineEmits, ref } from 'vue'
import { SetVBUSPower } from '../../wailsjs/go/main/App'
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

const emit = defineEmits(['close'])
const authStore = useAuthStore()
const logStore = useLogStore()
const uiStore = useUIStore()

const { executeWithAuth, markPasswordRejected } = authStore
const pending = ref(false)

const close = () => {
  if (pending.value) return
  emit('close')
}

const confirmVBUSPowerChange = async (enabled) => {
  const title = enabled ? 'Confirm Vbus Power On' : 'Confirm Vbus Power Off'
  const message = enabled
    ? 'Are you sure you want to set Vbus Power to "On"?'
    : 'When Vbus Power is set to "Off", you must connect an external power supply for the hub to operate properly.\n\nDo you want to continue?'

  return uiStore.showConfirm(message, title, {
    confirmLabel: 'Yes',
    cancelLabel: 'No'
  })
}

const sendVBUSPowerCommand = async (enabled, attempt = 0) => {
  if (!props.device?.portId) return

  const deviceId = props.device.portId
  const password = props.device.sessionPassword || CONSTANTS.DEFAULT_PASSWORD

  pending.value = true
  try {
    const resp = await SetVBUSPower(deviceId, password, enabled)
    const normalizedResp = String(resp || '').trim()

    if (normalizedResp.includes('E01')) {
      await markPasswordRejected(deviceId)

      if (attempt >= CONSTANTS.MAX_AUTH_RETRIES) {
        logStore.addLog('Error', 'VBUS power authentication failed', deviceId)
        uiStore.showAlert('Device authentication failed. Cannot switch VBUS power.', 'Error')
        return
      }

      executeWithAuth(async () => {
        await sendVBUSPowerCommand(enabled, attempt + 1)
      }, null, true)
      return
    }

    if (normalizedResp.startsWith('G')) {
      logStore.addLog('User Action', `VBUS power turned ${enabled ? 'on' : 'off'}`, deviceId)
      emit('close')
      return
    }

    logStore.addLog('Error', `VBUS power command failed: ${normalizedResp || 'empty response'}`, deviceId)
    uiStore.showAlert(`VBUS power command failed. Device responded: ${normalizedResp || 'empty response'}`, 'Error')
  } catch (e) {
    logStore.addLog('Error', 'VBUS power command failed: ' + e, deviceId)
    uiStore.showAlert('VBUS power command failed: ' + e, 'Error')
  } finally {
    pending.value = false
  }
}

const selectState = async (enabled) => {
  if (pending.value) return

  const confirmed = await confirmVBUSPowerChange(enabled)
  if (!confirmed) return

  executeWithAuth(async () => {
    await sendVBUSPowerCommand(enabled)
  })
}
</script>

<template>
  <div v-if="show" class="modal-overlay">
    <div class="modal">
      <div class="modal-header">
        <span>Vbus Power Option</span>
        <button
          class="header-close"
          type="button"
          :disabled="pending"
          aria-label="Close"
          @click="close"
        >&times;</button>
      </div>
      <div class="modal-body">
        <div class="modal-message">Please choose a Vbus Power Option.</div>
      </div>
      <div class="modal-footer action-row">
        <button @click="selectState(true)" :disabled="pending" class="primary">Vbus Power On</button>
        <button @click="selectState(false)" :disabled="pending">Vbus Power Off</button>
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
  width: 320px;
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
  padding: 18px 15px;
}

.modal-message {
  text-align: center;
  color: #333;
}

.modal-footer {
  padding: 10px;
  display: flex;
  gap: 10px;
  background-color: #f0f0f0;
}

.action-row {
  justify-content: center;
}

button {
  cursor: pointer;
  background-color: #e1e1e1;
  border: 1px solid #adadad;
  padding: 2px 16px;
  min-width: 72px;
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

button.primary {
  background-color: #0078d7;
  color: #fff;
  border-color: #005a9e;
}

button.primary:hover:not(:disabled) {
  background-color: #005a9e;
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
