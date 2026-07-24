import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { validateUserCapacity } from './user-config.js';

const BASE_URL = (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
const EVENT_ID = Number(__ENV.EVENT_ID || '1');
const SECTION_ID = Number(__ENV.SECTION_ID || '0');
const SECTION_IDS = parseSectionIDs(__ENV.SECTION_IDS || '');
const QUANTITY = Number(__ENV.QUANTITY || '1');
const VUS = Number(__ENV.VUS || '20');
const MAX_QUEUE_POLLS = Number(__ENV.MAX_QUEUE_POLLS || '30');
const QUEUE_POLL_SECONDS = Number(__ENV.QUEUE_POLL_SECONDS || '1');
const QUEUE_JOIN_RETRIES = Number(__ENV.QUEUE_JOIN_RETRIES || '5');
const QUEUE_JOIN_RETRY_SECONDS = Number(__ENV.QUEUE_JOIN_RETRY_SECONDS || '1');
const HOLD_MINUTES = Number(__ENV.HOLD_MINUTES || '10');
const RUN_ID = __ENV.RUN_ID || `${Date.now()}`;
const sessionFile = JSON.parse(open(__ENV.SESSION_FILE || './sessions.json'));

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
const queueJoinRetries = new Counter('queue_join_retries');

export function setup() {
  validateUserCapacity(VUS);
  validateSessionCapacity(VUS);

  const health = http.get(`${BASE_URL}/healthz`, requestOptions('healthz'));
  check(health, {
    'healthz is 200': (res) => res.status === 200,
  });

  const availability = http.get(`${BASE_URL}/events/${EVENT_ID}/availability`, requestOptions('availability'));
  check(availability, {
    'availability is 200': (res) => res.status === 200,
  });

  const saleStatus = getSaleStatus();
  if (!saleStatus) {
    throw new Error('Event is not ready for queue. Check sale status and event sale window.');
  }

  const sections = safeJson(availability) || [];
  let selectedSectionIds = [];
  if (SECTION_ID) {
    selectedSectionIds = [SECTION_ID];
  } else if (SECTION_IDS.length > 0) {
    selectedSectionIds = SECTION_IDS.filter((sectionId) => sectionIsAvailable(sections, sectionId));
  } else {
    selectedSectionIds = sections
      .filter((section) => Number(section.available_quantity) >= QUANTITY)
      .map((section) => Number(section.section_id))
      .filter((sectionId) => sectionId > 0);
  }

  if (selectedSectionIds.length === 0) {
    throw new Error('No available section. Set SECTION_ID or check event availability.');
  }

  return {
    sectionIds: selectedSectionIds,
  };
}

export default function (data) {
  const vuID = __VU;
  const iterationID = __ITER;
  const buyerKey = `${RUN_ID}-${vuID}-${iterationID}`;
  const session = sessionFile.sessions[vuID - 1];
  const sectionId = selectSectionId(data.sectionIds, vuID, iterationID);

  if (!session || !session.cookieHeader) {
    reservationSuccessRate.add(false);
    orderSuccessRate.add(false);
    bookingFlowSuccessRate.add(false);
    return;
  }

  const queueStatus = joinQueue(buyerKey, session.cookieHeader);
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

  const reservation = reserveTicket(sectionId, readyStatus.purchase_token, session.cookieHeader);
  reservationSuccessRate.add(Boolean(reservation && reservation.id));
  if (!reservation || !reservation.id) {
    orderSuccessRate.add(false);
    bookingFlowSuccessRate.add(false);
    return;
  }

  const order = createOrder(reservation.id, readyStatus.purchase_token, buyerKey, session.cookieHeader);
  orderSuccessRate.add(Boolean(order && order.id));
  bookingFlowSuccessRate.add(Boolean(order && order.id));

  sleep(0.1);
}

function getSaleStatus() {
  const res = http.get(`${BASE_URL}/sale/status?event_id=${EVENT_ID}`, requestOptions('sale_status'));

  check(res, {
    'sale status is 200': (response) => response.status === 200,
    'event can join queue': (response) => safeJsonField(response, 'can_join_queue') === true,
  });

  if (res.status !== 200 || safeJsonField(res, 'can_join_queue') !== true) {
    return null;
  }

  return safeJson(res);
}

function joinQueue(buyerKey, cookieHeader) {
	const payload = {
		event_id: EVENT_ID,
		client_id: `k6-client-${buyerKey}`,
		request_id: `k6-request-${buyerKey}`,
		channel: 'k6',
	};

	for (let attempt = 0; attempt <= QUEUE_JOIN_RETRIES; attempt += 1) {
		const res = http.post(`${BASE_URL}/queue/join`, JSON.stringify(payload), {
			...requestOptions('queue_join'),
			headers: authenticatedHeaders(cookieHeader),
		});

		check(res, {
			'queue join is 201': (response) => response.status === 201,
			'queue token exists': (response) => Boolean(safeJsonField(response, 'queue_token')),
		});

		if (res.status === 201) {
			return safeJson(res);
		}

		classifyError(res);
		if (!shouldRetryQueueJoin(res) || attempt === QUEUE_JOIN_RETRIES) {
			return null;
		}

		queueJoinRetries.add(1);
		sleep(queueJoinRetrySeconds(res));
	}

	return null;
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
    const res = http.get(`${BASE_URL}/queue/status/${status.queue_token}`, requestOptions('queue_status'));

    check(res, {
      'queue status is 200': (response) => response.status === 200,
    });

    if (res.status !== 200) {
      classifyError(res);
      return null;
    }

    status = safeJson(res);
  }

  queuePollsTrend.add(MAX_QUEUE_POLLS);
  return null;
}

function reserveTicket(sectionId, purchaseToken, cookieHeader) {
  const holdUntil = new Date(Date.now() + HOLD_MINUTES * 60 * 1000).toISOString();
  const payload = {
    event_id: EVENT_ID,
    section_id: sectionId,
    quantity: QUANTITY,
    hold_until: holdUntil,
    purchase_token: purchaseToken,
  };

  const res = http.post(`${BASE_URL}/reservations`, JSON.stringify(payload), {
    ...requestOptions('reserve'),
    headers: authenticatedHeaders(cookieHeader),
  });

  check(res, {
    'reservation is 201': (response) => response.status === 201,
    'reservation id exists': (response) => Boolean(safeJsonField(response, 'id')),
  });

  if (res.status !== 201) {
    classifyError(res);
    return null;
  }

  return safeJson(res);
}

function createOrder(reservationId, purchaseToken, buyerKey, cookieHeader) {
  const payload = {
    reservation_id: reservationId,
    order_no: `K6-${buyerKey}`.slice(0, 40),
    purchase_token: purchaseToken,
  };

  const res = http.post(`${BASE_URL}/orders`, JSON.stringify(payload), {
    ...requestOptions('create_order'),
    headers: authenticatedHeaders(cookieHeader),
  });

  check(res, {
    'order is 201': (response) => response.status === 201,
    'order id exists': (response) => Boolean(safeJsonField(response, 'id')),
  });

  if (res.status !== 201) {
    classifyError(res);
    return null;
  }

  return safeJson(res);
}

function classifyError(res) {
  const message = String(safeJsonField(res, 'message') || safeJsonField(res, 'error') || '');

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

function safeJsonField(res, fieldName) {
  try {
    return res.json(fieldName);
  } catch {
    return '';
  }
}

function safeJson(res) {
	try {
		return res.json();
	} catch {
		return null;
	}
}

function shouldRetryQueueJoin(res) {
	return res.status === 0 || res.status === 429 || res.status >= 500;
}

function queueJoinRetrySeconds(res) {
	const retryAfter = Number((res.headers && res.headers['Retry-After']) || '0');
	if (retryAfter > 0) {
		return retryAfter;
	}
	return QUEUE_JOIN_RETRY_SECONDS;
}

function parseSectionIDs(value) {
  return value
    .split(',')
    .map((sectionId) => Number(sectionId.trim()))
    .filter((sectionId) => sectionId > 0);
}

function sectionIsAvailable(sections, sectionId) {
  return sections.some((section) => (
    Number(section.section_id) === sectionId
    && Number(section.available_quantity) >= QUANTITY
  ));
}

function selectSectionId(sectionIds, vuID, iterationID) {
  return sectionIds[(vuID + iterationID - 1) % sectionIds.length];
}

function authenticatedHeaders(cookieHeader) {
  return {
    ...jsonHeaders,
    Cookie: cookieHeader,
  };
}

function requestOptions(name) {
  return {
    tags: { name },
  };
}

function validateSessionCapacity(requiredSessions) {
  if (!sessionFile.sessions || sessionFile.sessions.length < requiredSessions) {
    const actualSessions = sessionFile.sessions ? sessionFile.sessions.length : 0;
    throw new Error(`Need ${requiredSessions} sessions, but sessions.json only has ${actualSessions}. Run tests/k6/login-users.mjs first.`);
  }
}
