<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import Button from 'primevue/button'
import Tag from 'primevue/tag'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { useBookingFlowStore } from '../stores/bookingFlow'
import type { EventResponse } from '../types/api'

const flow = useBookingFlowStore()

const displayEvents = computed(() => {
  if (flow.events.length === 0) return []
  const repeated = [...flow.events]
  while (repeated.length < 8) {
    repeated.push(...flow.events)
  }
  return repeated.slice(0, 8).map((event, index) => ({ ...event, displayId: `${event.id}-${index}` }))
})

function formatDate(value: string) {
  return new Date(value).toLocaleDateString('zh-TW', {
    month: '2-digit',
    day: '2-digit',
    weekday: 'short',
  })
}

function posterClass(index: number) {
  return `poster poster-${(index % 6) + 1}`
}

function shortTitle(event: EventResponse) {
  return event.name.length > 34 ? `${event.name.slice(0, 34)}...` : event.name
}

async function reload() {
  try {
    await flow.loadEvents()
  } catch {}
}

onMounted(reload)
</script>

<template>
  <AppShell>
    <section class="category-title">
      <span></span>
      <h1>Concerts</h1>
      <span></span>
    </section>

    <StateBanner
      :loading="flow.isPending('events')"
      :error="flow.getError('events')"
      loading-text="Loading events..."
      :retryable="true"
      @retry="reload"
    />

    <section class="event-grid">
      <RouterLink
        v-for="(event, index) in displayEvents"
        :key="event.displayId"
        class="event-card"
        :to="`/events/${event.id}`"
      >
        <div :class="posterClass(index)">
          <small>{{ formatDate(event.start_at) }}</small>
          <strong>{{ event.name }}</strong>
          <span>{{ event.venue }}</span>
        </div>
        <h2>{{ shortTitle(event) }}</h2>
        <Tag :value="event.status === 1 ? 'On sale' : 'Preparing'" severity="danger" />
      </RouterLink>
    </section>

    <div v-if="flow.events.length === 0 && !flow.isPending('events')" class="empty-panel">
      <p>No events from backend yet.</p>
      <Button label="Reload" icon="pi pi-refresh" @click="reload" />
    </div>
  </AppShell>
</template>
