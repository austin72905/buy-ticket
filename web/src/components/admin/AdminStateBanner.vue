<script setup lang="ts">
import StateBanner from '../StateBanner.vue'

defineProps<{
  loading?: boolean
  error?: string
  loadingText?: string
  retryable?: boolean
}>()

defineEmits<{
  retry: []
}>()
</script>

<template>
  <StateBanner
    :loading="loading"
    :error="error"
    :loading-text="loadingText"
    :retryable="retryable"
    retry-label="Retry"
    @retry="$emit('retry')"
  />
</template>
