<script setup>
import { defineProps, defineEmits } from 'vue'

const props = defineProps({
  devices: {
    type: Array,
    default: () => []
  },
  currentDeviceId: {
    type: String,
    default: ''
  },
  isScanning: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['select', 'scan'])

const onSelectChange = (e) => {
  emit('select', e.target.value)
}
</script>

<template>
  <div class="hub-selection">
    <label>Selected USB Hub</label>
    <div class="selection-row">
      <select :value="currentDeviceId" @change="onSelectChange">
        <option v-if="!currentDeviceId" value="">Select a hub...</option>
        <option v-for="dev in devices" :key="dev.portId" :value="dev.portId">
          {{ dev.displayName }}
        </option>
      </select>
      <button @click="$emit('scan')" :disabled="isScanning">Scan for USB Hubs</button>
    </div>
  </div>
</template>

<style scoped>
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
  border: 1px solid #ccc;
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
