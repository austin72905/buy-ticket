<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import Tag from 'primevue/tag'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { useBookingFlowStore } from '../stores/bookingFlow'

const route = useRoute()
const router = useRouter()
const flow = useBookingFlowStore()

const eventId = computed(() => Number(route.params.eventId))
let pollingTimer: number | undefined

const ready = computed(() => Boolean(flow.purchaseToken))

async function load() {
  try {
    await flow.selectEvent(eventId.value)
  } catch {}
}

function stopPolling() {
  if (pollingTimer) {
    window.clearInterval(pollingTimer)
    pollingTimer = undefined
  }
}

function startPolling() {
  stopPolling()
  pollingTimer = window.setInterval(async () => {
    if (!flow.queueToken || flow.purchaseToken) {
      stopPolling()
      return
    }
    try {
      await flow.refreshQueueStatus()
    } catch {}
  }, 1500)
}

async function enterQueue() {
  try {
    await flow.joinQueueAction({
      clientId: `web-${flow.userId}`,
      requestId: `req-${Date.now()}`,
      channel: 'web',
    })
    if (!flow.purchaseToken) startPolling()
  } catch {}
}

function goTickets() {
  router.push(`/events/${eventId.value}/tickets`)
}

watch(
  () => flow.purchaseToken,
  (token) => {
    if (token) {
      stopPolling()
      goTickets()
    }
  },
)

onMounted(load)
onBeforeUnmount(stopPolling)
</script>

<template>
  <AppShell>
    <StateBanner
      :loading="flow.isPending('eventBundle') || flow.isPending('queueJoin') || flow.isPending('queueStatus')"
      :error="flow.getError('eventBundle') || flow.getError('queueJoin') || flow.getError('queueStatus')"
      loading-text="Processing..."
      :retryable="true"
      @retry="flow.queueToken ? flow.refreshQueueStatus() : load()"
    />

    <section v-if="flow.currentEvent" class="notice-page">
      <div class="detail-heading">
        <span></span>
        <h1>{{ flow.currentEvent.name }}</h1>
        <span></span>
      </div>

      <div class="notice-box">
        <ul>
          <li>Complete member login before purchasing. This demo uses user id {{ flow.userId }}.</li>
          <li>After entering the queue, the backend returns a queue token and may return a purchase token.</li>
          <li>When purchase token is ready, the page moves to seat and quantity selection.</li>
          <li>Reservation is a temporary hold. Order payment must finish before it expires.</li>
          <li>Use Chrome or a modern browser for the smoothest checkout flow.</li>
        </ul>

        <div v-if="flow.queueStatus" class="queue-card">
          <div>
            <span>Queue Status</span>
            <Tag :value="String(flow.queueStatus.status)" :severity="ready ? 'success' : 'warning'" />
          </div>
          <div>
            <span>Ahead</span>
            <strong>{{ flow.queueStatus.ahead_count }}</strong>
          </div>
          <div>
            <span>Token</span>
            <strong class="mono">{{ flow.queueToken }}</strong>
          </div>
        </div>

        <div class="center-actions">
          <Button
            v-if="!ready"
            label="Start Queue"
            severity="danger"
            size="large"
            :loading="flow.isPending('queueJoin') || flow.isPending('queueStatus')"
            @click="enterQueue"
          />
          <Button v-else label="Select Tickets" severity="danger" size="large" @click="goTickets" />
        </div>
      </div>
    </section>
  </AppShell>
</template>
