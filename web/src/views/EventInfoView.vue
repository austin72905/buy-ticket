<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import Tag from 'primevue/tag'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { QueueStatus, normalizeQueueStatus } from '../lib/statusValues'
import { useBookingFlowStore } from '../stores/bookingFlow'

const route = useRoute()
const router = useRouter()
const flow = useBookingFlowStore()

const eventId = computed(() => Number(route.params.eventId))
let pollingTimer: number | undefined

const queueStatus = computed(() => (flow.queueStatus ? normalizeQueueStatus(flow.queueStatus.status) : ''))
const waiting = computed(() => queueStatus.value === QueueStatus.Waiting)
const ready = computed(() => queueStatus.value === QueueStatus.Ready && Boolean(flow.purchaseToken))
const expired = computed(() => queueStatus.value === QueueStatus.Expired)
const canStartQueue = computed(() => !flow.queueStatus || expired.value)
const queueStatusLabel = computed(() => {
  if (!flow.queueStatus) return 'Not Joined'
  if (waiting.value) return 'Waiting'
  if (queueStatus.value === QueueStatus.Ready) return ready.value ? 'Ready' : 'Ready - Missing Token'
  if (expired.value) return 'Expired'
  return `Unknown (${queueStatus.value})`
})
const queueStatusSeverity = computed(() => {
  if (ready.value) return 'success'
  if (expired.value) return 'danger'
  return 'warning'
})

async function load() {
  try {
    await flow.selectEvent(eventId.value)
    if (waiting.value) {
      startPolling()
    }
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
    if (!flow.queueToken || ready.value || expired.value) {
      stopPolling()
      return
    }
    try {
      await flow.refreshQueueStatus()
    } catch {}
  }, 1500)
}

async function enterQueue() {
  if (!flow.currentUser) {
    router.push({
      name: 'login',
      query: {
        redirect: route.fullPath,
      },
    })
    return
  }

  try {
    await flow.joinQueueAction({
      clientId: `web-${flow.userId}`,
      requestId: `req-${Date.now()}`,
      channel: 'web',
    })
    if (waiting.value) startPolling()
  } catch {}
}

function goTickets() {
  router.push(`/events/${eventId.value}/tickets`)
}

watch(
  () => [flow.queueStatus?.status, flow.purchaseToken],
  () => {
    if (ready.value) {
      stopPolling()
      goTickets()
      return
    }

    if (expired.value) {
      stopPolling()
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
          <li>After entering the queue, the backend returns a queue token with waiting status.</li>
          <li>The page polls queue status until the scheduler releases a purchase token.</li>
          <li>When status is ready and purchase token exists, the page moves to seat and quantity selection.</li>
          <li>If queue status expires, start queue again.</li>
          <li>Reservation is a temporary hold. Order payment must finish before it expires.</li>
          <li>Use Chrome or a modern browser for the smoothest checkout flow.</li>
        </ul>

        <div v-if="flow.queueStatus" class="queue-card">
          <div>
            <span>Queue Status</span>
            <Tag :value="queueStatusLabel" :severity="queueStatusSeverity" />
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

        <p v-if="waiting" class="small-muted">
          You are waiting in queue. This page will keep polling until you are released.
        </p>
        <p v-if="expired" class="small-muted">
          Queue token expired. Start queue again to get a new queue token.
        </p>

        <div class="center-actions">
          <Button
            v-if="canStartQueue"
            label="Start Queue"
            severity="danger"
            size="large"
            :loading="flow.isPending('queueJoin') || flow.isPending('queueStatus')"
            @click="enterQueue"
          />
          <Button
            v-else-if="waiting"
            label="Waiting In Queue"
            severity="secondary"
            size="large"
            disabled
            :loading="flow.isPending('queueStatus')"
          />
          <Button v-else-if="ready" label="Select Tickets" severity="danger" size="large" @click="goTickets" />
          <Button
            v-else
            label="Waiting For Purchase Token"
            severity="secondary"
            size="large"
            disabled
            :loading="flow.isPending('queueStatus')"
          />
        </div>
      </div>
    </section>
  </AppShell>
</template>
