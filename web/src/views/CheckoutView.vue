<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import Button from 'primevue/button'
import Card from 'primevue/card'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Select from 'primevue/select'
import Tag from 'primevue/tag'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { useBookingFlowStore } from '../stores/bookingFlow'

const flow = useBookingFlowStore()

const quantity = ref(1)
const holdMinutes = ref(10)
const orderNo = ref(`ORD-${Date.now()}`)
const paymentNo = ref(`PAY-${Date.now()}`)
const paymentMethod = ref('credit_card')

const totalAmount = computed(() =>
  flow.selectedSection ? flow.selectedSection.price * quantity.value : 0,
)

const canReserve = computed(
  () => Boolean(flow.selectedEventId && flow.selectedSectionId && flow.purchaseToken),
)
const canCreateOrder = computed(() => Boolean(flow.reservation?.id))
const canPay = computed(() => Boolean(flow.order?.id))

async function loadSections() {
  try {
    await flow.refreshSections()
  } catch {}
}

async function onReserve() {
  try {
    await flow.reserveTicketAction({
      quantity: quantity.value,
      holdMinutes: holdMinutes.value,
    })
  } catch {}
}

async function onCreateOrder() {
  try {
    await flow.createOrderAction({
      orderNo: orderNo.value,
      holdMinutes: holdMinutes.value,
    })
  } catch {}
}

async function onFetchOrder() {
  try {
    await flow.fetchOrderAction()
  } catch {}
}

async function onPay() {
  try {
    await flow.payOrderAction({
      paymentNo: paymentNo.value,
      method: paymentMethod.value,
    })
  } catch {}
}

async function onRetry() {
  if (flow.getError('sections')) {
    await loadSections()
    return
  }

  if (canPay.value) {
    await onPay()
    return
  }

  if (canCreateOrder.value) {
    await onCreateOrder()
    return
  }

  await onReserve()
}

watch(
  () => flow.selectedEventId,
  async (eventId) => {
    if (eventId) {
      await loadSections()
    }
  },
)

onMounted(async () => {
  if (flow.selectedEventId) {
    await loadSections()
  }
})
</script>

<template>
  <AppShell>
    <section class="stack">
      <Message severity="secondary">
        Test `POST /reservations`, `POST /orders`, and `POST /payments` after queue returns a purchase token.
      </Message>

      <StateBanner
        :loading="
          flow.isPending('sections') ||
          flow.isPending('reserve') ||
          flow.isPending('createOrder') ||
          flow.isPending('fetchOrder') ||
          flow.isPending('payOrder')
        "
        :error="
          flow.getError('sections') ||
          flow.getError('reserve') ||
          flow.getError('createOrder') ||
          flow.getError('fetchOrder') ||
          flow.getError('payOrder')
        "
        loading-text="Calling checkout API..."
        :retryable="true"
        @retry="onRetry"
      />

      <div class="grid-two">
        <Card>
          <template #title>Reserve Ticket</template>
          <template #content>
            <div class="form-grid">
              <div class="field-inline">
                <label>event_id</label>
                <strong>{{ flow.selectedEventId ?? 'No event selected' }}</strong>
              </div>
              <div class="field-inline">
                <label for="sectionId">section_id</label>
                <Select
                  id="sectionId"
                  v-model="flow.selectedSectionId"
                  :options="flow.sections"
                  option-label="name"
                  option-value="id"
                  placeholder="Select section"
                />
              </div>
              <div class="field-inline">
                <label for="quantity">quantity</label>
                <InputNumber id="quantity" v-model="quantity" :min="1" :max="10" />
              </div>
              <div class="field-inline">
                <label for="holdMinutes">hold minutes</label>
                <InputNumber id="holdMinutes" v-model="holdMinutes" :min="1" :max="30" />
              </div>
              <div class="field-inline">
                <label>purchase_token</label>
                <strong class="mono">{{ flow.purchaseToken || 'Missing purchase token' }}</strong>
              </div>
              <div class="field-inline">
                <label>total</label>
                <Tag :value="String(totalAmount)" severity="info" />
              </div>
              <Button label="Create reservation" icon="pi pi-ticket" :disabled="!canReserve" :loading="flow.isPending('reserve')" @click="onReserve" />
            </div>
          </template>
        </Card>

        <Card>
          <template #title>Reservation Result</template>
          <template #content>
            <div v-if="flow.reservation" class="detail-list">
              <div><span>reservation_id</span><strong>{{ flow.reservation.id }}</strong></div>
              <div><span>status</span><Tag :value="String(flow.reservation.status)" severity="info" /></div>
              <div><span>total_amount</span><strong>{{ flow.reservation.total_amount }}</strong></div>
              <div><span>expires_at</span><strong>{{ flow.reservation.expires_at }}</strong></div>
            </div>
            <p v-else class="empty-copy">Reservation response appears here after a successful hold.</p>
          </template>
        </Card>
      </div>

      <div class="grid-two">
        <Card>
          <template #title>Create Order</template>
          <template #content>
            <div class="form-grid">
              <div class="field-inline">
                <label for="orderNo">order_no</label>
                <InputText id="orderNo" v-model="orderNo" />
              </div>
              <Button label="Create order" icon="pi pi-shopping-cart" :disabled="!canCreateOrder" :loading="flow.isPending('createOrder')" @click="onCreateOrder" />
              <Button label="Fetch order" severity="secondary" icon="pi pi-search" :disabled="!canPay" :loading="flow.isPending('fetchOrder')" @click="onFetchOrder" />
            </div>
          </template>
        </Card>

        <Card>
          <template #title>Order Result</template>
          <template #content>
            <div v-if="flow.order" class="detail-list">
              <div><span>order_id</span><strong>{{ flow.order.id }}</strong></div>
              <div><span>order_no</span><strong class="mono">{{ flow.order.order_no }}</strong></div>
              <div><span>status</span><Tag :value="String(flow.order.status)" severity="warning" /></div>
              <div><span>total_amount</span><strong>{{ flow.order.total_amount }}</strong></div>
            </div>
            <p v-else class="empty-copy">Order response appears here after reservation is converted.</p>
          </template>
        </Card>
      </div>

      <div class="grid-two">
        <Card>
          <template #title>Pay Order</template>
          <template #content>
            <div class="form-grid">
              <div class="field-inline">
                <label for="paymentNo">payment_no</label>
                <InputText id="paymentNo" v-model="paymentNo" />
              </div>
              <div class="field-inline">
                <label for="paymentMethod">method</label>
                <InputText id="paymentMethod" v-model="paymentMethod" />
              </div>
              <Button label="Pay order" icon="pi pi-credit-card" :disabled="!canPay" :loading="flow.isPending('payOrder')" @click="onPay" />
            </div>
          </template>
        </Card>

        <Card>
          <template #title>Payment Result</template>
          <template #content>
            <div v-if="flow.payment" class="detail-list">
              <div><span>payment_id</span><strong>{{ flow.payment.id }}</strong></div>
              <div><span>payment_no</span><strong class="mono">{{ flow.payment.payment_no }}</strong></div>
              <div><span>status</span><Tag :value="String(flow.payment.status)" severity="success" /></div>
              <div><span>amount</span><strong>{{ flow.payment.amount }}</strong></div>
            </div>
            <p v-else class="empty-copy">Payment response appears here after a successful pay API call.</p>
          </template>
        </Card>
      </div>
    </section>
  </AppShell>
</template>
