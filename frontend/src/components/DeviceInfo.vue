<script setup>
import { defineProps } from 'vue'

const props = defineProps({
  device: {
    type: Object,
    default: null
  }
})
</script>

<template>
  <div class="hub-info" v-if="device">
    <div class="info-text">
      <div class="model-number">C2G54464</div>
      <div class="description">C2G 7-Port USB-A Hub</div>
    </div>
    <div class="hub-image">
      <!-- Placeholder for Hub Image -->
      <div class="img-placeholder">Hub Image</div>
    </div>
  </div>
</template>

<style scoped>
.hub-info {
  display: flex;
  gap: 10px;
  align-items: center;
}

.hub-image .img-placeholder {
  width: 100px;
  height: 30px;
  background-color: #333;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  border-radius: 2px;
}
</style>
