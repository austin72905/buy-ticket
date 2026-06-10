<script setup lang="ts">
import Button from 'primevue/button'
import Message from 'primevue/message'
import ProgressSpinner from 'primevue/progressspinner'

defineProps<{
  loading?: boolean
  error?: string
  loadingText?: string
  retryable?: boolean
  retryLabel?: string
}>()

defineEmits<{
  retry: []
}>()
</script>

<template>
  <div v-if="loading || error" class="state-banner">
    <Message v-if="loading" severity="secondary">
      <div class="state-banner__content">
        <ProgressSpinner
          stroke-width="6"
          style="width: 1.1rem; height: 1.1rem"
          animation-duration=".8s"
        />
        <span>{{ loadingText || 'Loading...' }}</span>
      </div>
    </Message>
    <Message v-if="error" severity="error">
      <div class="state-banner__content state-banner__content--split">
        <span>{{ error }}</span>
        <Button
          v-if="retryable"
          :label="retryLabel || 'Retry'"
          size="small"
          severity="danger"
          outlined
          @click="$emit('retry')"
        />
      </div>
    </Message>
  </div>
</template>
