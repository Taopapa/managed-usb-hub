<script setup>
import { defineProps, defineEmits, ref, watch } from 'vue'

const DEFAULT_PASSWORD = 'pass    '

const props = defineProps({
  show: Boolean,
  initialOld: String
})

const emit = defineEmits(['close', 'submit'])

const oldPass = ref('')
const newPass = ref('')
const confirmPass = ref('')

watch(() => props.show, (newVal) => {
  if (newVal) {
    oldPass.value = props.initialOld || DEFAULT_PASSWORD
    newPass.value = DEFAULT_PASSWORD
    confirmPass.value = DEFAULT_PASSWORD
  }
})

const submit = () => {
  emit('submit', {
    old: oldPass.value || props.initialOld || DEFAULT_PASSWORD,
    new: newPass.value,
    confirm: confirmPass.value
  })
}

const cancel = () => {
  emit('close')
}
</script>

<template>
  <div class="modal-overlay" v-if="show">
    <div class="modal">
      <div class="modal-header">Change Password</div>
      <div class="modal-body form-body">
        <p style="font-size: 12px; margin-top: 0; margin-bottom: 5px; color: #666;">Password must be 3-8 characters.</p>
        <div class="form-group">
          <label>Old Password:</label>
          <input type="password" v-model="oldPass" maxlength="8">
        </div>
        <div class="form-group">
          <label>New Password:</label>
          <input type="password" v-model="newPass" maxlength="8">
        </div>
        <div class="form-group">
          <label>Confirm Password:</label>
          <input type="password" v-model="confirmPass" maxlength="8">
        </div>
      </div>
      <div class="modal-footer">
        <button @click="cancel">Cancel</button>
        <button @click="submit">Update</button>
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

.modal-footer {
  padding: 10px;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  background-color: #f0f0f0;
}

input {
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
