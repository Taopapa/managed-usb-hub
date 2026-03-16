<script setup>
import { defineProps } from 'vue'

const props = defineProps({
  portStates: {
    type: Object,
    default: () => ({})
  },
  totalPorts: {
    type: Number,
    default: 7
  }
})
</script>

<template>
  <div class="ports-table-container">
    <table class="data-table">
      <thead>
        <tr>
          <th>Port</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="n in totalPorts" :key="n">
          <td>{{ n }}</td>
          <td>{{ portStates[n] ? 'Enabled' : 'Disabled' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.ports-table-container {
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
</style>
