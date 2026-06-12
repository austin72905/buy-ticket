import { createRouter, createWebHistory } from 'vue-router'

import CheckoutView from '../views/CheckoutView.vue'
import EventDetailView from '../views/EventDetailView.vue'
import EventInfoView from '../views/EventInfoView.vue'
import EventsView from '../views/EventsView.vue'
import OrderView from '../views/OrderView.vue'
import PaymentView from '../views/PaymentView.vue'
import ReservationView from '../views/ReservationView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/events',
    },
    {
      path: '/events',
      name: 'events',
      component: EventsView,
    },
    {
      path: '/events/:eventId',
      name: 'event-detail',
      component: EventDetailView,
    },
    {
      path: '/events/:eventId/info',
      name: 'event-info',
      component: EventInfoView,
    },
    {
      path: '/events/:eventId/tickets',
      name: 'ticket-select',
      component: CheckoutView,
    },
    {
      path: '/events/:eventId/reservation',
      name: 'reservation-review',
      component: ReservationView,
    },
    {
      path: '/events/:eventId/order',
      name: 'order-review',
      component: OrderView,
    },
    {
      path: '/events/:eventId/payment',
      name: 'payment',
      component: PaymentView,
    },
  ],
})

export default router
