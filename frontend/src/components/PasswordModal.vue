<script setup>
import { defineProps, defineEmits, ref, watch } from 'vue'

const props = defineProps({
  show: Boolean,
  initialPassword: String
})

const emit = defineEmits(['close', 'submit'])

const password = ref('')

// Focus input when modal opens
// Simple way: watch show prop
watch(() => props.show, (newVal) => {
  if (newVal) {
    password.value = props.initialPassword || ''
    // Focus logic would need a ref to the input element and nextTick
  }
})

const submit = () => {
  emit('submit', password.value || props.initialPassword || '')
}

const cancel = () => {
  emit('close')
}
</script>

<template>
  <div class="modal-overlay" v-if="show">
    <div class="modal">
      <div class="modal-header">Authentication Required</div>
      <div class="modal-body">
        <p>Enter Password (3-8 chars):</p>
        <input 
          type="password" 
          v-model="password" 
          maxlength="8" 
          @keyup.enter="submit"
          autofocus
        >
      </div>
      <div class="modal-footer">
        <button @click="cancel">Cancel</button>
        <button @click="submit">Confirm</button>
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
  z-index: 10001;
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
}

.modal-body {
  padding: 15px;
}

.modal-footer {
  padding: 10px;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  background-color: #f0f0f0;
}

input {
  width: 100%;
  border: 1px solid #ccc;
  padding: 2px;
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
</style>
