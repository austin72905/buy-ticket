<script setup lang="ts">
import Button from 'primevue/button'

defineProps<{
  hasMore?: boolean
  loading?: boolean
}>()

defineEmits<{
  loadMore: []
}>()
</script>

<template>
  <div v-if="hasMore" class="admin-load-more">
    <Button label="Load More" severity="secondary" :loading="loading" @click="$emit('loadMore')" />
  </div>
</template>
