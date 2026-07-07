import { createRouter, createWebHistory } from 'vue-router'

import AdminAuditLogsView from '../views/admin/AdminAuditLogsView.vue'
import AdminEventDetailView from '../views/admin/AdminEventDetailView.vue'
import AdminEventsView from '../views/admin/AdminEventsView.vue'
import AdminForbiddenView from '../views/admin/AdminForbiddenView.vue'
import AdminLoginView from '../views/admin/AdminLoginView.vue'
import AdminOrdersView from '../views/admin/AdminOrdersView.vue'
import AdminOrganizersView from '../views/admin/AdminOrganizersView.vue'
import AdminUsersView from '../views/admin/AdminUsersView.vue'
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
import { useAdminAuthStore } from '../stores/adminAuth'
import { useBookingFlowStore } from '../stores/bookingFlow'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/events',
    },
    {
      path: '/admin',
      redirect: '/admin/orders',
    },
    {
      path: '/admin/login',
      name: 'admin-login',
      component: AdminLoginView,
    },
    {
      path: '/admin/orders',
      name: 'admin-orders',
      component: AdminOrdersView,
      meta: { requiresAdminAuth: true },
    },
    {
      path: '/admin/events',
      name: 'admin-events',
      component: AdminEventsView,
      meta: { requiresAdminAuth: true },
    },
    {
      path: '/admin/events/:eventId',
      name: 'admin-event-detail',
      component: AdminEventDetailView,
      meta: { requiresAdminAuth: true },
    },
    {
      path: '/admin/audit-logs',
      name: 'admin-audit-logs',
      component: AdminAuditLogsView,
      meta: { requiresAdminAuth: true },
    },
    {
      path: '/admin/users',
      name: 'admin-users',
      component: AdminUsersView,
      meta: { requiresAdminAuth: true },
    },
    {
      path: '/admin/organizers',
      name: 'admin-organizers',
      component: AdminOrganizersView,
      meta: { requiresAdminAuth: true },
    },
    {
      path: '/admin/forbidden',
      name: 'admin-forbidden',
      component: AdminForbiddenView,
      meta: { requiresAdminAuth: true },
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
  if (to.matched.some((record) => record.meta.requiresAdminAuth)) {
    const adminAuth = useAdminAuthStore()

    if (adminAuth.currentAdmin) {
      return true
    }

    try {
      await adminAuth.loadAdminMe()
      return true
    } catch {
      adminAuth.clearAdmin()
      return {
        name: 'admin-login',
        query: {
          redirect: to.fullPath,
        },
      }
    }
  }

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
