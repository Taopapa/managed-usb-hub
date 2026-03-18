<script setup>
import { defineProps, defineEmits } from 'vue'

const props = defineProps({
  show: Boolean,
  title: {
    type: String,
    default: 'Confirm'
  },
  message: String
})

const emit = defineEmits(['result'])

const handleResult = (result) => {
  emit('result', result)
}
</script>

<template>
  <div class="modal-overlay" v-if="show" style="z-index: 9999;">
    <div class="modal alert-modal">
      <div class="modal-header">{{ title }}</div>
      <div class="modal-body alert-message">
        {{ message }}
      </div>
      <div class="modal-footer" style="justify-content: center; gap: 20px;">
        <button @click="handleResult(false)" style="min-width: 80px;">Cancel</button>
        <button @click="handleResult(true)" style="min-width: 80px;" class="primary">OK</button>
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
  z-index: 1000;
}

.modal {
  background-color: #fff;
  border: 1px solid #999;
  box-shadow: 2px 2px 10px rgba(0, 0, 0, 0.2);
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

button.primary {
  background-color: #0078d7;
  color: white;
  border-color: #005a9e;
}

button.primary:hover {
  background-color: #005a9e;
}

.alert-modal {
  max-width: 400px;
  text-align: center;
}

.alert-message {
  margin: 15px 0;
  white-space: pre-wrap;
  color: #495057;
  font-size: 14px;
}
</style>