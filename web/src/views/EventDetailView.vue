<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { useBookingFlowStore } from '../stores/bookingFlow'

const route = useRoute()
const router = useRouter()
const flow = useBookingFlowStore()

const eventId = computed(() => Number(route.params.eventId))
const minPrice = computed(() => Math.min(...flow.sections.map((section) => section.price)))
const maxPrice = computed(() => Math.max(...flow.sections.map((section) => section.price)))

function formatDateTime(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-TW', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatCurrency(value: number) {
  return new Intl.NumberFormat('zh-TW', {
    style: 'currency',
    currency: 'TWD',
    maximumFractionDigits: 0,
  }).format(value)
}

async function load() {
  try {
    await flow.selectEvent(eventId.value)
  } catch {}
}

function goInfo() {
  router.push(`/events/${eventId.value}/info`)
}

onMounted(load)
</script>

<template>
  <AppShell>
    <StateBanner
      :loading="flow.isPending('eventBundle')"
      :error="flow.getError('eventBundle')"
      loading-text="Loading event..."
      :retryable="true"
      @retry="load"
    />

    <section v-if="flow.currentEvent" class="event-detail">
      <div class="detail-heading">
        <span></span>
        <h1>{{ flow.currentEvent.name }}</h1>
        <span></span>
      </div>

      <div class="event-hero">
        <div class="hero-poster">
          <small>{{ formatDateTime(flow.currentEvent.sale_start_at) }} sale</small>
          <strong>{{ flow.currentEvent.name }}</strong>
          <span>{{ flow.currentEvent.venue }}</span>
        </div>
      </div>

      <p class="sale-note">
        Please confirm event rules and ticket notes before entering the queue. Completing a purchase means
        you agree to the current event policy and checkout rules.
      </p>

      <div class="center-actions">
        <Button label="Buy Tickets" severity="danger" size="large" @click="goInfo" />
      </div>

      <nav class="info-tabs">
        <a class="active">Event intro</a>
        <a>Ticket notes</a>
        <a>Payment</a>
        <a>Pickup</a>
        <a>Refund</a>
      </nav>

      <section class="detail-copy">
        <h2>Event Information</h2>
        <div class="info-grid">
          <div>
            <span>Date and Time</span>
            <strong>{{ formatDateTime(flow.currentEvent.start_at) }}</strong>
          </div>
          <div>
            <span>Location</span>
            <strong>{{ flow.currentEvent.venue }}</strong>
          </div>
          <div>
            <span>Sale Window</span>
            <strong>{{ formatDateTime(flow.currentEvent.sale_start_at) }}</strong>
          </div>
          <div>
            <span>Price</span>
            <strong v-if="Number.isFinite(minPrice)">{{ formatCurrency(minPrice) }} - {{ formatCurrency(maxPrice) }}</strong>
            <strong v-else>-</strong>
          </div>
        </div>
      </section>

      <section class="ticket-table">
        <div class="table-row table-head">
          <span>Session</span>
          <span>Location</span>
          <span>Price</span>
          <span>Action</span>
        </div>
        <div v-for="section in flow.sections" :key="section.id" class="table-row">
          <span>{{ formatDateTime(flow.currentEvent.start_at) }}</span>
          <span>{{ flow.currentEvent.venue }} / {{ section.name }}</span>
          <span>{{ formatCurrency(section.price) }}</span>
          <Button label="Buy" severity="danger" size="small" @click="goInfo" />
        </div>
      </section>
    </section>
  </AppShell>
</template>
