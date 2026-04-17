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
    <div class="hub-image">
      <img src="../static/product.png" alt="Hub Image" class="product-image" />
    </div>
    <div class="info-text">
      <div class="model-number">{{ device.deviceName || 'C2G 7-port Managed USB HUB' }}</div>
    </div>
  </div>
</template>

<style scoped>
.hub-info {
  display: flex;
  gap: 10px;
  align-items: center;
  width: 45%;
}

.hub-image .product-image {
  width: 100px;
  height: auto;
  border-radius: 2px;
  margin-right: 20px;
}
</style>
