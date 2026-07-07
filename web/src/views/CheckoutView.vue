<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import InputNumber from 'primevue/inputnumber'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { SectionStatus, normalizeSectionStatus } from '../lib/statusValues'
import { useBookingFlowStore } from '../stores/bookingFlow'
import type { SectionAvailabilityResponse } from '../types/api'

const route = useRoute()
const router = useRouter()
const flow = useBookingFlowStore()

const eventId = computed(() => Number(route.params.eventId))
const quantity = ref(1)
const holdMinutes = ref(10)

const selectedAvailability = computed(
  () => flow.availability.find((item) => item.section_id === flow.selectedSectionId) ?? null,
)
const selectedSection = computed(
  () => flow.sections.find((section) => section.id === flow.selectedSectionId) ?? null,
)
const selectedPurchaseLimit = computed(() => selectedSection.value?.purchase_limit ?? 0)
const selectedQuantityLimit = computed(() => {
  if (!selectedAvailability.value || !selectedSection.value) return 0
  return Math.min(selectedAvailability.value.available_quantity, selectedSection.value.purchase_limit)
})
const quantityInputMax = computed(() => Math.max(1, selectedQuantityLimit.value))
const totalAmount = computed(() => (selectedSection.value ? selectedSection.value.price * quantity.value : 0))
const selectedSectionActive = computed(() =>
  selectedAvailability.value ? isSectionSelectable(selectedAvailability.value) : false,
)
const canReserve = computed(() =>
  Boolean(
    flow.purchaseToken &&
      flow.selectedSectionId &&
      selectedSectionActive.value &&
      selectedQuantityLimit.value > 0 &&
      quantity.value > 0 &&
      quantity.value <= selectedQuantityLimit.value,
  ),
)

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

function availableBySection(sectionId: number) {
  return flow.availability.find((item) => item.section_id === sectionId)?.available_quantity ?? 0
}

function isSectionActive(section: SectionAvailabilityResponse) {
  const fullSection = flow.sections.find((item) => item.id === section.section_id)
  return (
    normalizeSectionStatus(section.status) === SectionStatus.Active &&
    (!fullSection || normalizeSectionStatus(fullSection.status) === SectionStatus.Active)
  )
}

function isSectionSelectable(section: SectionAvailabilityResponse) {
  return isSectionActive(section) && section.available_quantity > 0
}

function selectSection(section: SectionAvailabilityResponse) {
  if (!isSectionSelectable(section)) return
  flow.selectedSectionId = section.section_id
}

async function load() {
  try {
    await flow.selectEvent(eventId.value)
  } catch {}
}

async function reserve() {
  if (!canReserve.value) return

  try {
    await flow.reserveTicketAction({
      quantity: quantity.value,
      holdMinutes: holdMinutes.value,
    })
    if (flow.reservation) {
      router.push(`/events/${eventId.value}/reservation`)
    }
  } catch {}
}

function backToQueue() {
  router.push(`/events/${eventId.value}/info`)
}

watch([selectedQuantityLimit, selectedSectionActive], () => {
  if (!selectedSectionActive.value || selectedQuantityLimit.value <= 0) {
    quantity.value = 1
    return
  }

  if (quantity.value > selectedQuantityLimit.value) {
    quantity.value = selectedQuantityLimit.value
  }

  if (quantity.value < 1) {
    quantity.value = 1
  }
})

onMounted(load)
</script>

<template>
  <AppShell>
    <StateBanner
      :loading="
        flow.isPending('eventBundle') ||
        flow.isPending('reserve')
      "
      :error="
        flow.getError('eventBundle') ||
        flow.getError('reserve')
      "
      loading-text="Holding selected tickets..."
      :retryable="true"
      @retry="load"
    />

    <section v-if="flow.currentEvent" class="ticket-page">
      <div class="detail-heading">
        <span></span>
        <h1>{{ flow.currentEvent.name }}</h1>
        <span></span>
      </div>

      <div class="event-summary">
        <div>
          <strong>Event Date</strong>
          <span>{{ formatDateTime(flow.currentEvent.start_at) }}</span>
        </div>
        <div>
          <strong>Location</strong>
          <span>{{ flow.currentEvent.venue }}</span>
        </div>
        <Button label="Change Event" severity="danger" size="small" @click="router.push('/events')" />
      </div>

      <div v-if="!flow.purchaseToken" class="empty-panel">
        <p>Queue purchase token is required before selecting tickets.</p>
        <Button label="Go Queue" severity="danger" @click="backToQueue" />
      </div>

      <div v-else class="seat-layout">
        <section class="map-panel">
          <div class="map-tools">
            <span><i class="pi pi-search-plus"></i> Zoom in</span>
            <span><i class="pi pi-search-minus"></i> Zoom out</span>
          </div>

          <div class="stage">Stage</div>
          <div class="seat-map">
            <button
              v-for="section in flow.availability"
              :key="section.section_id"
              :class="[
                'seat-block',
                `seat-color-${(section.section_id % 5) + 1}`,
                {
                  selected: flow.selectedSectionId === section.section_id,
                  disabled: !isSectionSelectable(section),
                },
              ]"
              :disabled="!isSectionSelectable(section)"
              @click="selectSection(section)"
            >
              {{ section.name }}
            </button>
          </div>
        </section>

        <aside class="ticket-panel">
          <div class="ticket-panel-head">
            <span>Color</span>
            <span>Area</span>
            <span>Price</span>
            <span>Left</span>
          </div>

          <button
            v-for="section in flow.availability"
            :key="section.section_id"
            :class="['ticket-row', { active: flow.selectedSectionId === section.section_id }]"
            :disabled="!isSectionSelectable(section)"
            @click="selectSection(section)"
          >
            <span :class="['color-dot', `seat-color-${(section.section_id % 5) + 1}`]"></span>
            <span>{{ section.name }}</span>
            <strong>{{ formatCurrency(section.price) }}</strong>
            <span>
              {{
                !isSectionActive(section)
                  ? 'Inactive'
                  : section.available_quantity > 0
                    ? section.available_quantity
                    : 'Sold out'
              }}
            </span>
          </button>

          <div class="order-box">
            <div class="order-line">
              <span>Selected</span>
              <strong>{{ selectedAvailability?.name ?? '-' }}</strong>
            </div>
            <div class="order-line">
              <span>Quantity</span>
              <InputNumber
                v-model="quantity"
                :min="1"
                :max="quantityInputMax"
                :disabled="!selectedSectionActive || selectedQuantityLimit <= 0"
                show-buttons
                button-layout="horizontal"
              />
            </div>
            <div class="order-line">
              <span>Limit</span>
              <strong>
                {{ selectedPurchaseLimit > 0 ? `每人限購 ${selectedPurchaseLimit} 張` : '-' }}
              </strong>
            </div>
            <div class="order-line">
              <span>Total</span>
              <strong>{{ formatCurrency(totalAmount) }}</strong>
            </div>
            <Button
              label="Hold Ticket"
              severity="danger"
              :disabled="!canReserve"
              :loading="flow.isPending('reserve')"
              @click="reserve"
            />
          </div>

          <p class="small-muted">
            Backend connected: queue token, purchase token, and reservation hold.
            Available left: {{ selectedAvailability ? availableBySection(selectedAvailability.section_id) : '-' }}
          </p>
          <p v-if="selectedAvailability" class="small-muted">
            Max selectable quantity:
            {{ selectedQuantityLimit > 0 ? selectedQuantityLimit : '-' }}
          </p>
        </aside>
      </div>
    </section>
  </AppShell>
</template>
