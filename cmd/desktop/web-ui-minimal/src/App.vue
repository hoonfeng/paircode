<template>
  <div>
    <h1>Vue 3 Test</h1>
    <p>{{ message }}</p>
  </div>
</template>

<script setup>
import { ref } from 'vue'
const message = ref('Hello from wb-ui!')
</script>
