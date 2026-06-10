import { createRouter, createWebHistory } from 'vue-router'

import CheckoutView from '../views/CheckoutView.vue'
import EventsView from '../views/EventsView.vue'
import QueueView from '../views/QueueView.vue'

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
      path: '/queue',
      name: 'queue',
      component: QueueView,
    },
    {
      path: '/checkout',
      name: 'checkout',
      component: CheckoutView,
    },
  ],
})

export default router
