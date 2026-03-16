<script setup>
import { defineProps, watch, onMounted } from 'vue'
import { useLogStore } from '../stores/logs'
import { useUIStore } from '../stores/ui'
import { storeToRefs } from 'pinia'

const props = defineProps({
  currentDeviceId: {
    type: String,
    default: ''
  }
})

const logStore = useLogStore()
const uiStore = useUIStore()

const { logs } = storeToRefs(logStore)
const { loadHistoryLogs, clearLogs } = logStore
const { showAlert } = uiStore

// Watch currentDeviceId to reload logs
watch(() => props.currentDeviceId, (newVal) => {
    loadHistoryLogs(newVal)
})

onMounted(() => {
    loadHistoryLogs(props.currentDeviceId)
})

const handleClear = async () => {
    if (confirm(`Are you sure you want to clear logs for ${props.currentDeviceId || 'System'}?`)) {
        try {
            await clearLogs(props.currentDeviceId)
        } catch (e) {
            showAlert("Failed to clear logs: " + e, "Error")
        }
    }
}

const handleExport = () => {
    const csvContent = "data:text/csv;charset=utf-8," 
        + logs.value.map(e => `${e.time},${e.event},${e.deviceID},${e.details}`).join("\n");
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement("a");
    link.setAttribute("href", encodedUri);
    
    const deviceName = props.currentDeviceId || "System"
    const fileName = `hub_logs_${deviceName}_${new Date().toISOString().split('T')[0]}.csv`
    
    link.setAttribute("download", fileName);
    document.body.appendChild(link);
    link.click();
}
</script>

<template>
  <div class="logs-container">
    <div class="logs-toolbar">
      <button @click="handleClear">Clear Logs</button>
      <button @click="handleExport">Export Logs</button>
    </div>
    <div class="logs-table-wrapper">
      <table class="data-table">
        <thead>
          <tr>
            <th style="width: 150px;">Time</th>
            <th style="width: 100px;">Device</th>
            <th>Details</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(log, index) in logs" :key="index">
            <td>{{ log.time }}</td>
            <td>{{ log.deviceID }}</td>
            <td>{{ log.details }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.logs-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.logs-toolbar {
  padding: 5px;
  background-color: #f9f9f9;
  border-bottom: 1px solid #ddd;
  display: flex;
  gap: 10px;
}

.logs-table-wrapper {
  flex: 1;
  overflow: auto;
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

.data-table tr:nth-child(even) {
  background-color: #f9f9f9;
}

.data-table tr:hover {
  background-color: #eef;
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
