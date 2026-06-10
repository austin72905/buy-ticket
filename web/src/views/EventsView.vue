<script setup lang="ts">
import { onMounted } from 'vue'
import Button from 'primevue/button'
import Card from 'primevue/card'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import Select from 'primevue/select'
import Tag from 'primevue/tag'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { useBookingFlowStore } from '../stores/bookingFlow'

const flow = useBookingFlowStore()

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}

function formatCurrency(value: number) {
  return new Intl.NumberFormat('zh-TW', {
    style: 'currency',
    currency: 'TWD',
    maximumFractionDigits: 0,
  }).format(value)
}

async function onEventChange(eventId: number | null) {
  if (!eventId) return
  try {
    await flow.selectEvent(eventId)
  } catch {}
}

async function reloadCurrentEvent() {
  try {
    if (flow.selectedEventId) {
      await flow.loadEventBundle(flow.selectedEventId)
      return
    }

    await flow.loadEvents()
  } catch {}
}

onMounted(async () => {
  try {
    await flow.loadEvents()

    if (flow.selectedEventId) {
      await flow.selectEvent(flow.selectedEventId)
    }
  } catch {}
})
</script>

<template>
  <AppShell>
    <section class="stack">
      <div class="toolbar">
        <div class="field-inline grow">
          <label for="eventId">event_id</label>
          <Select
            id="eventId"
            :model-value="flow.selectedEventId"
            :options="flow.events"
            option-label="name"
            option-value="id"
            placeholder="Select event"
            class="full-width"
            :loading="flow.isPending('events')"
            @update:model-value="onEventChange"
          />
        </div>
        <Button
          label="Reload event bundle"
          icon="pi pi-refresh"
          :loading="flow.isPending('eventBundle')"
          @click="reloadCurrentEvent"
        />
      </div>

      <StateBanner
        :loading="flow.isPending('events') || flow.isPending('eventBundle')"
        :error="flow.getError('events') || flow.getError('eventBundle')"
        loading-text="Loading events and availability..."
        :retryable="true"
        @retry="reloadCurrentEvent"
      />

      <div v-if="flow.currentEvent" class="grid-two">
        <Card>
          <template #title>{{ flow.currentEvent.name }}</template>
          <template #subtitle>{{ flow.currentEvent.venue }}</template>
          <template #content>
            <div class="detail-list">
              <div><span>sale_start_at</span><strong>{{ formatDate(flow.currentEvent.sale_start_at) }}</strong></div>
              <div><span>sale_end_at</span><strong>{{ formatDate(flow.currentEvent.sale_end_at) }}</strong></div>
              <div><span>start_at</span><strong>{{ formatDate(flow.currentEvent.start_at) }}</strong></div>
            </div>
          </template>
        </Card>

        <Card>
          <template #title>Sale Status</template>
          <template #content>
            <div v-if="flow.saleStatus" class="detail-list">
              <div><span>is_on_sale</span><Tag :severity="flow.saleStatus.is_on_sale ? 'success' : 'danger'" :value="String(flow.saleStatus.is_on_sale)" /></div>
              <div><span>can_join_queue</span><Tag :severity="flow.saleStatus.can_join_queue ? 'success' : 'warning'" :value="String(flow.saleStatus.can_join_queue)" /></div>
              <div><span>can_reserve</span><Tag :severity="flow.saleStatus.can_reserve ? 'success' : 'warning'" :value="String(flow.saleStatus.can_reserve)" /></div>
              <div><span>server_time</span><strong>{{ formatDate(flow.saleStatus.server_time) }}</strong></div>
            </div>
          </template>
        </Card>
      </div>

      <div class="grid-two">
        <Card>
          <template #title>Sections</template>
          <template #content>
            <DataTable :value="flow.sections" size="small" data-key="id">
              <Column field="name" header="Section" />
              <Column header="Price">
                <template #body="{ data }">
                  {{ formatCurrency(data.price) }}
                </template>
              </Column>
              <Column field="purchase_limit" header="Limit" />
              <Column field="reserved_quantity" header="Reserved" />
              <Column field="sold_quantity" header="Sold" />
            </DataTable>
          </template>
        </Card>

        <Card>
          <template #title>Availability</template>
          <template #content>
            <DataTable :value="flow.availability" size="small" data-key="section_id">
              <Column field="name" header="Section" />
              <Column header="Price">
                <template #body="{ data }">
                  {{ formatCurrency(data.price) }}
                </template>
              </Column>
              <Column field="available_quantity" header="Available" />
              <Column field="reserved_quantity" header="Reserved" />
              <Column field="sold_quantity" header="Sold" />
            </DataTable>
          </template>
        </Card>
      </div>
    </section>
  </AppShell>
</template>
