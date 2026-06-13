import { createRouter, createWebHistory } from 'vue-router'

import CheckoutView from '../views/CheckoutView.vue'
import EventDetailView from '../views/EventDetailView.vue'
import EventInfoView from '../views/EventInfoView.vue'
import EventsView from '../views/EventsView.vue'
import LoginView from '../views/LoginView.vue'
import MyOrdersView from '../views/MyOrdersView.vue'
import OrderView from '../views/OrderView.vue'
import PaymentView from '../views/PaymentView.vue'
import RegisterView from '../views/RegisterView.vue'
import ReservationView from '../views/ReservationView.vue'
import { useBookingFlowStore } from '../stores/bookingFlow'

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
      path: '/login',
      name: 'login',
      component: LoginView,
    },
    {
      path: '/register',
      name: 'register',
      component: RegisterView,
    },
    {
      path: '/orders',
      name: 'my-orders',
      component: MyOrdersView,
      meta: { requiresAuth: true },
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
      meta: { requiresAuth: true },
    },
    {
      path: '/events/:eventId/tickets',
      name: 'ticket-select',
      component: CheckoutView,
      meta: { requiresAuth: true },
    },
    {
      path: '/events/:eventId/reservation',
      name: 'reservation-review',
      component: ReservationView,
      meta: { requiresAuth: true },
    },
    {
      path: '/events/:eventId/order',
      name: 'order-review',
      component: OrderView,
      meta: { requiresAuth: true },
    },
    {
      path: '/events/:eventId/payment',
      name: 'payment',
      component: PaymentView,
      meta: { requiresAuth: true },
    },
  ],
})

router.beforeEach(async (to) => {
  if (!to.matched.some((record) => record.meta.requiresAuth)) {
    return true
  }

  const flow = useBookingFlowStore()

  if (flow.currentUser) {
    return true
  }

  try {
    await flow.loadMe()
    return true
  } catch {
    return {
      name: 'login',
      query: {
        redirect: to.fullPath,
      },
    }
  }
})

export default router
