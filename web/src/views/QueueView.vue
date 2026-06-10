<script setup lang="ts">
import { computed, ref } from 'vue'
import Button from 'primevue/button'
import Card from 'primevue/card'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Tag from 'primevue/tag'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { useBookingFlowStore } from '../stores/bookingFlow'

const flow = useBookingFlowStore()

const clientId = ref('web-device-001')
const requestId = ref(`req-${Date.now()}`)
const channel = ref('web')
const accessCode = ref('')

const canJoin = computed(() => Boolean(flow.selectedEventId))

async function onJoinQueue() {
  try {
    await flow.joinQueueAction({
      clientId: clientId.value,
      requestId: requestId.value,
      channel: channel.value,
      accessCode: accessCode.value || undefined,
    })
  } catch {}
}

async function onRefreshStatus() {
  try {
    await flow.refreshQueueStatus()
  } catch {}
}
</script>

<template>
  <AppShell>
    <section class="stack">
      <Message severity="secondary">
        Test `POST /queue/join` and `GET /queue/status/:queueToken` against the current backend.
      </Message>

      <StateBanner
        :loading="flow.isPending('queueJoin') || flow.isPending('queueStatus')"
        :error="flow.getError('queueJoin') || flow.getError('queueStatus')"
        loading-text="Calling queue API..."
        :retryable="true"
        @retry="flow.queueToken ? onRefreshStatus() : onJoinQueue()"
      />

      <div class="grid-two">
        <Card>
          <template #title>Join Queue</template>
          <template #content>
            <div class="form-grid">
              <div class="field-inline">
                <label>event_id</label>
                <strong>{{ flow.selectedEventId ?? 'No event selected' }}</strong>
              </div>
              <div class="field-inline">
                <label>user_id</label>
                <strong>{{ flow.userId }}</strong>
              </div>
              <div class="field-inline">
                <label for="clientId">client_id</label>
                <InputText id="clientId" v-model="clientId" />
              </div>
              <div class="field-inline">
                <label for="requestId">request_id</label>
                <InputText id="requestId" v-model="requestId" />
              </div>
              <div class="field-inline">
                <label for="channel">channel</label>
                <InputText id="channel" v-model="channel" />
              </div>
              <div class="field-inline">
                <label for="accessCode">access_code</label>
                <InputText id="accessCode" v-model="accessCode" />
              </div>
              <Button label="Join queue" icon="pi pi-send" :disabled="!canJoin" :loading="flow.isPending('queueJoin')" @click="onJoinQueue" />
            </div>
          </template>
        </Card>

        <Card>
          <template #title>Queue Snapshot</template>
          <template #content>
            <div v-if="flow.queueStatus" class="detail-list">
              <div><span>status</span><Tag :value="String(flow.queueStatus.status)" severity="info" /></div>
              <div><span>queue_token</span><strong class="mono">{{ flow.queueStatus.queue_token }}</strong></div>
              <div><span>queue_position</span><strong>{{ flow.queueStatus.queue_position }}</strong></div>
              <div><span>ahead_count</span><strong>{{ flow.queueStatus.ahead_count }}</strong></div>
              <div><span>purchase_token</span><strong class="mono">{{ flow.queueStatus.purchase_token ?? 'Not ready yet' }}</strong></div>
              <div><span>expires_at</span><strong>{{ flow.queueStatus.expired_at }}</strong></div>
              <Button label="Refresh status" icon="pi pi-refresh" :loading="flow.isPending('queueStatus')" @click="onRefreshStatus" />
            </div>
            <p v-else class="empty-copy">Join the queue first to capture a queue token and purchase token.</p>
          </template>
        </Card>
      </div>
    </section>
  </AppShell>
</template>
