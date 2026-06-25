import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

const BASE_URL = (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
const EVENT_ID = Number(__ENV.EVENT_ID || '1');
const SECTION_ID = Number(__ENV.SECTION_ID || '0');
const QUANTITY = Number(__ENV.QUANTITY || '1');
const VUS = Number(__ENV.VUS || '20');
const MAX_QUEUE_POLLS = Number(__ENV.MAX_QUEUE_POLLS || '30');
const QUEUE_POLL_SECONDS = Number(__ENV.QUEUE_POLL_SECONDS || '1');
const HOLD_MINUTES = Number(__ENV.HOLD_MINUTES || '10');
const RUN_ID = __ENV.RUN_ID || `${Date.now()}`;

const jsonHeaders = {
  'Content-Type': 'application/json',
};

export const options = {
  scenarios: {
    booking_flow: {
      executor: 'per-vu-iterations',
      vus: VUS,
      iterations: 1,
      maxDuration: __ENV.MAX_DURATION || '2m',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<1000', 'p(99)<2000'],
    booking_flow_success_rate: ['rate>0.95'],
    reservation_success_rate: ['rate>0.95'],
    order_success_rate: ['rate>0.95'],
  },
};

const bookingFlowSuccessRate = new Rate('booking_flow_success_rate');
const reservationSuccessRate = new Rate('reservation_success_rate');
const orderSuccessRate = new Rate('order_success_rate');
const queueReadyRate = new Rate('queue_ready_rate');
const queuePollsTrend = new Trend('queue_polls');
const activeReservationErrors = new Counter('active_reservation_errors');
const purchaseTokenErrors = new Counter('purchase_token_errors');
const sectionStockErrors = new Counter('section_stock_errors');

export function setup() {
  const health = http.get(`${BASE_URL}/healthz`);
  check(health, {
    'healthz is 200': (res) => res.status === 200,
  });

  const availability = http.get(`${BASE_URL}/events/${EVENT_ID}/availability`);
  check(availability, {
    'availability is 200': (res) => res.status === 200,
  });

  let selectedSectionId = SECTION_ID;
  if (!selectedSectionId) {
    const sections = availability.json() || [];
    const availableSection = sections.find((section) => Number(section.available_quantity) >= QUANTITY);
    selectedSectionId = Number((availableSection && availableSection.section_id) || 0);
  }

  if (!selectedSectionId) {
    throw new Error('No available section. Set SECTION_ID or check event availability.');
  }

  return {
    sectionId: selectedSectionId,
  };
}

export default function (data) {
  const vuID = __VU;
  const iterationID = __ITER;
  const buyerKey = `${RUN_ID}-${vuID}-${iterationID}`;
  const email = `k6-${buyerKey}@load.local`;
  const password = 'password123';
  const sectionId = data.sectionId;

  const user = registerOrLogin(email, password, buyerKey);
  if (!user) {
    reservationSuccessRate.add(false);
    orderSuccessRate.add(false);
    bookingFlowSuccessRate.add(false);
    return;
  }

  const saleStatus = getSaleStatus();
  if (!saleStatus) {
    reservationSuccessRate.add(false);
    orderSuccessRate.add(false);
    bookingFlowSuccessRate.add(false);
    return;
  }

  const queueStatus = joinQueue(buyerKey);
  if (!queueStatus || !queueStatus.queue_token) {
    reservationSuccessRate.add(false);
    orderSuccessRate.add(false);
    bookingFlowSuccessRate.add(false);
    return;
  }

  const readyStatus = waitForPurchaseToken(queueStatus);
  queueReadyRate.add(Boolean(readyStatus && readyStatus.purchase_token));
  if (!readyStatus || !readyStatus.purchase_token) {
    reservationSuccessRate.add(false);
    orderSuccessRate.add(false);
    bookingFlowSuccessRate.add(false);
    return;
  }

  const reservation = reserveTicket(sectionId, readyStatus.purchase_token);
  reservationSuccessRate.add(Boolean(reservation && reservation.id));
  if (!reservation || !reservation.id) {
    orderSuccessRate.add(false);
    bookingFlowSuccessRate.add(false);
    return;
  }

  const order = createOrder(reservation.id, readyStatus.purchase_token, buyerKey);
  orderSuccessRate.add(Boolean(order && order.id));
  bookingFlowSuccessRate.add(Boolean(order && order.id));

  sleep(0.1);
}

function registerOrLogin(email, password, buyerKey) {
  const registerPayload = {
    name: `K6 Buyer ${buyerKey}`,
    email,
    password,
  };

  const registerRes = http.post(`${BASE_URL}/auth/register`, JSON.stringify(registerPayload), {
    headers: jsonHeaders,
  });

  if (registerRes.status === 201 || registerRes.status === 200) {
    check(registerRes, {
      'register returns user': (res) => Boolean(res.json('id')),
    });
    return registerRes.json();
  }

  const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({ email, password }), {
    headers: jsonHeaders,
  });

  check(loginRes, {
    'login is 200': (res) => res.status === 200,
  });

  if (loginRes.status !== 200) {
    return null;
  }

  return loginRes.json();
}

function getSaleStatus() {
  const res = http.get(`${BASE_URL}/sale/status?event_id=${EVENT_ID}`);

  check(res, {
    'sale status is 200': (response) => response.status === 200,
    'event can join queue': (response) => response.json('can_join_queue') === true,
  });

  if (res.status !== 200 || res.json('can_join_queue') !== true) {
    return null;
  }

  return res.json();
}

function joinQueue(buyerKey) {
  const payload = {
    event_id: EVENT_ID,
    client_id: `k6-client-${buyerKey}`,
    request_id: `k6-request-${buyerKey}`,
    channel: 'k6',
  };

  const res = http.post(`${BASE_URL}/queue/join`, JSON.stringify(payload), {
    headers: jsonHeaders,
  });

  check(res, {
    'queue join is 201': (response) => response.status === 201,
    'queue token exists': (response) => Boolean(response.json('queue_token')),
  });

  if (res.status !== 201) {
    classifyError(res);
    return null;
  }

  return res.json();
}

function waitForPurchaseToken(initialStatus) {
  let status = initialStatus;

  for (let poll = 0; poll <= MAX_QUEUE_POLLS; poll += 1) {
    if (status && status.purchase_token) {
      queuePollsTrend.add(poll);
      return status;
    }

    if (poll === MAX_QUEUE_POLLS) {
      break;
    }

    sleep(QUEUE_POLL_SECONDS);
    const res = http.get(`${BASE_URL}/queue/status/${status.queue_token}`);

    check(res, {
      'queue status is 200': (response) => response.status === 200,
    });

    if (res.status !== 200) {
      classifyError(res);
      return null;
    }

    status = res.json();
  }

  queuePollsTrend.add(MAX_QUEUE_POLLS);
  return null;
}

function reserveTicket(sectionId, purchaseToken) {
  const holdUntil = new Date(Date.now() + HOLD_MINUTES * 60 * 1000).toISOString();
  const payload = {
    event_id: EVENT_ID,
    section_id: sectionId,
    quantity: QUANTITY,
    hold_until: holdUntil,
    purchase_token: purchaseToken,
  };

  const res = http.post(`${BASE_URL}/reservations`, JSON.stringify(payload), {
    headers: jsonHeaders,
  });

  check(res, {
    'reservation is 201': (response) => response.status === 201,
    'reservation id exists': (response) => Boolean(response.json('id')),
  });

  if (res.status !== 201) {
    classifyError(res);
    return null;
  }

  return res.json();
}

function createOrder(reservationId, purchaseToken, buyerKey) {
  const payload = {
    reservation_id: reservationId,
    order_no: `K6-${buyerKey}`.slice(0, 40),
    purchase_token: purchaseToken,
  };

  const res = http.post(`${BASE_URL}/orders`, JSON.stringify(payload), {
    headers: jsonHeaders,
  });

  check(res, {
    'order is 201': (response) => response.status === 201,
    'order id exists': (response) => Boolean(response.json('id')),
  });

  if (res.status !== 201) {
    classifyError(res);
    return null;
  }

  return res.json();
}

function classifyError(res) {
  const message = String(jsonField(res, 'error') || '');

  if (message.includes('active reservation already exists')) {
    activeReservationErrors.add(1);
  }
  if (message.includes('purchase token')) {
    purchaseTokenErrors.add(1);
  }
  if (message.includes('section cannot reserve requested quantity')) {
    sectionStockErrors.add(1);
  }
}

function jsonField(res, fieldName) {
  try {
    return res.json(fieldName);
  } catch {
    return '';
  }
}
