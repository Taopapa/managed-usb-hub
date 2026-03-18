<script setup>
import { defineProps, defineEmits, ref, watch, inject } from 'vue'
import { GetScheduledTasks, AddScheduledTask, UpdateScheduledTask, RemoveScheduledTask } from '../../wailsjs/go/main/App'
import { parseMaskToStates, statesToMask } from '../utils/maskUtils'
import { useDeviceStore } from '../stores/devices'
import { useUIStore } from '../stores/ui'

const deviceStore = useDeviceStore()
const uiStore = useUIStore()

const props = defineProps({
  show: Boolean
})

const emit = defineEmits(['close'])
const { showAlert, showConfirm } = uiStore

const tasks = ref([])
const isEditingTask = ref(false)
const taskForm = ref({
  id: '',
  deviceID: '',
  daysOfWeek: [],
  startTime: '08:00',
  stopTime: '17:00',
  startPortStates: Array(16).fill(true),
  stopPortStates: Array(16).fill(false),
  enabled: true
})

const daysOptions = [
  { label: 'Sun', value: 0 },
  { label: 'Mon', value: 1 },
  { label: 'Tue', value: 2 },
  { label: 'Wed', value: 3 },
  { label: 'Thu', value: 4 },
  { label: 'Fri', value: 5 },
  { label: 'Sat', value: 6 },
]

const fetchTasks = async () => {
    try {
        const t = await GetScheduledTasks()
        tasks.value = t || []
    } catch(e) {
        console.error("Failed to fetch tasks", e)
    }
}

watch(() => props.show, (val) => {
    if (val) {
        fetchTasks()
        resetTaskForm()
    }
})

const resetTaskForm = () => {
  isEditingTask.value = false
  taskForm.value = {
    id: '',
    deviceID: deviceStore.devices.length > 0 ? deviceStore.devices[0].portId : '',
    daysOfWeek: [1, 2, 3, 4, 5],
    startTime: '08:00',
    stopTime: '17:00',
    startPortStates: Array(16).fill(true),
    stopPortStates: Array(16).fill(false),
    enabled: true
  }
}

const editTask = (task) => {
  isEditingTask.value = true
  
  taskForm.value = {
    id: task.id,
    deviceID: task.device_id,
    daysOfWeek: task.days_of_week || [],
    startTime: task.start_time,
    stopTime: task.stop_time,
    startPortStates: parseMaskToStates(task.start_mask || "FFFFFFFF"),
    stopPortStates: parseMaskToStates(task.stop_mask || "00000000"),
    enabled: task.enabled
  }
}

const saveTask = async () => {
    if (!taskForm.value.deviceID) {
        if(showAlert) showAlert("Device ID is required", "Validation Error")
        else alert("Device ID is required")
        return
    }

    const startMask = statesToMask(taskForm.value.startPortStates)
    const stopMask = statesToMask(taskForm.value.stopPortStates)
    
    const payload = {
        id: isEditingTask.value ? taskForm.value.id : Date.now().toString(),
        device_id: taskForm.value.deviceID,
        days_of_week: taskForm.value.daysOfWeek.map(Number),
        start_time: taskForm.value.startTime,
        stop_time: taskForm.value.stopTime,
        enabled: taskForm.value.enabled,
        start_mask: startMask,
        stop_mask: stopMask
    }

    try {
        if (isEditingTask.value) {
            await UpdateScheduledTask(payload)
        } else {
            await AddScheduledTask(payload)
        }
        await fetchTasks()
        resetTaskForm()
    } catch(e) {
        if(showAlert) showAlert("Failed to save task: " + e, "Error")
        else alert("Failed to save task: " + e)
    }
}

const deleteTask = async (id) => {
  const confirmed = await showConfirm("Delete this task?", "Confirm Delete")
  if (!confirmed) return
  try {
      await RemoveScheduledTask(id)
      await fetchTasks()
      if (isEditingTask.value && taskForm.value.id === id) resetTaskForm()
  } catch(e) {
      if(showAlert) showAlert("Failed to delete task: " + e, "Error")
      else alert("Failed to delete task: " + e)
  }
}

const formatDays = (days) => {
  if (!days || days.length === 0) return 'None'
  if (days.length === 7) return 'Every Day'
  const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
  // Sort days
  const sorted = [...days].sort((a,b) => a-b)
  return sorted.map(d => dayNames[d]).join(', ')
}
</script>

<template>
  <div v-if="show" class="modal-overlay">
    <div class="modal schedule-modal">
      <div class="modal-header">
        <span>Schedule Tasks</span>
        <button class="close-btn" @click="$emit('close')">×</button>
      </div>
      <div class="modal-body" style="display: flex; flex-direction: column;">
        <!-- Task List -->
        <div class="task-list">
          <table class="data-table">
            <thead>
              <tr>
                <th>Device</th>
                <th>Days</th>
                <th>On Time</th>
                <th>Off Time</th>
                <th>Active</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="task in tasks" :key="task.id" :class="{ 'editing-row': isEditingTask && taskForm.id === task.id }">
                <td>{{ task.device_id }}</td>
                <td>{{ formatDays(task.days_of_week) }}</td>
                <td>{{ task.start_time }}</td>
                <td>{{ task.stop_time }}</td>
                <td>{{ task.enabled ? 'Yes' : 'No' }}</td>
                <td>
                  <button @click="editTask(task)">Edit</button>
                  <button @click="deleteTask(task.id)">Del</button>
                </td>
              </tr>
              <tr v-if="tasks.length === 0">
                <td colspan="6" style="text-align: center;">No scheduled tasks</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Editor Form -->
        <div class="schedule-form">
          <div style="font-weight: bold; margin-bottom: 5px;">{{ isEditingTask ? 'Edit Task' : 'Add New Task' }}</div>
          
          <div class="form-row">
            <div class="form-group" style="flex: 1;">
              <label>Device ID (Port):</label>
              <select v-model="taskForm.deviceID" style="padding: 2px;">
                <option disabled value="">Select a device</option>
                <option v-for="dev in deviceStore.devices" :key="dev.portId" :value="dev.portId">
                  {{ dev.displayName || dev.portId }}
                </option>
              </select>
            </div>
            <div class="form-group">
              <label>Enabled:</label>
              <input type="checkbox" v-model="taskForm.enabled" style="margin-top: 5px;">
            </div>
          </div>
          
          <div class="form-group">
            <label>Days:</label>
            <div class="days-checkboxes">
              <label v-for="day in daysOptions" :key="day.value">
                <input type="checkbox" :value="day.value" v-model="taskForm.daysOfWeek">
                {{ day.label }}
              </label>
            </div>
          </div>
          
          <div class="form-row">
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
              <div class="port-mask-container">
                  <div v-for="(state, idx) in taskForm.startPortStates.slice(0, 7)" :key="'s'+idx" 
                        class="port-mask-item"
                        @click="taskForm.startPortStates[idx] = !state">
                      <div class="port-mask-box" :class="{ active: state }">
                          {{ idx + 1 }}
                      </div>
                  </div>
              </div>
          </div>

          <div class="form-group" style="margin-top: 5px;">
              <label>Stop Action Ports (When task stops):</label>
              <div class="port-mask-container">
                  <div v-for="(state, idx) in taskForm.stopPortStates.slice(0, 7)" :key="'e'+idx" 
                        class="port-mask-item"
                        @click="taskForm.stopPortStates[idx] = !state">
                      <div class="port-mask-box" :class="{ active: state }">
                          {{ idx + 1 }}
                      </div>
                  </div>
              </div>
              <div style="font-size: 10px; color: #666; margin-top: 2px;">* Click numbers to toggle port state. Green = On, Grey = Off.</div>
          </div>

          <div class="form-actions">
            <button v-if="isEditingTask" @click="resetTaskForm">Cancel Edit</button>
            <button @click="saveTask">{{ isEditingTask ? 'Update Task' : 'Add Task' }}</button>
          </div>
        </div>
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
  display: flex;
  flex-direction: column;
}

.schedule-modal {
  width: 700px;
  height: 600px;
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
  cursor: pointer;
}

.modal-body {
  padding: 15px;
  flex: 1;
  overflow: auto;
}

.task-list {
  flex: 1;
  min-height: 160px;
  overflow: auto;
  border: 1px solid #ccc;
  margin-bottom: 10px;
}

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

.editing-row {
  background-color: #e0f0ff;
}

.schedule-form {
  background-color: #f9f9f9;
  padding: 10px;
  border: 1px solid #ddd;
}

.form-row {
  display: flex;
  gap: 10px;
  margin-bottom: 5px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.days-checkboxes {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.days-checkboxes label {
  font-weight: normal;
  display: flex;
  align-items: center;
  gap: 2px;
}

.form-actions {
  margin-top: 10px;
  display: flex;
  gap: 5px;
  justify-content: flex-end;
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

input {
  border: 1px solid #ccc;
  padding: 2px;
}

.port-mask-container {
    display: flex; 
    gap: 5px; 
    background-color: #eee; 
    padding: 5px; 
    border-radius: 4px;
}

.port-mask-item {
    display: flex; 
    flex-direction: column; 
    align-items: center; 
    cursor: pointer;
}

.port-mask-box {
    width: 30px; 
    height: 30px; 
    background-color: #ccc; 
    border-radius: 4px; 
    display: flex; 
    align-items: center; 
    justify-content: center; 
    color: white; 
    font-weight: bold;
}

.port-mask-box.active {
    background-color: #4CAF50;
}
</style>
