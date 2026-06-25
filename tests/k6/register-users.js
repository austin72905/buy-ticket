import http from 'k6/http';
import { check } from 'k6';
import exec from 'k6/execution';
import { Counter, Rate } from 'k6/metrics';
import { configuredUserCount, userFromNumber, userOffset } from './user-config.js';

const BASE_URL = (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
const USER_COUNT = configuredUserCount();
const REGISTER_VUS = Number(__ENV.REGISTER_VUS || '50');

const jsonHeaders = {
  'Content-Type': 'application/json',
};

export const options = {
  scenarios: {
    register_users: {
      executor: 'shared-iterations',
      vus: REGISTER_VUS,
      iterations: USER_COUNT,
      maxDuration: __ENV.MAX_DURATION || '10m',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.10'],
    register_user_success_rate: ['rate>0.99'],
  },
};

const registerUserSuccessRate = new Rate('register_user_success_rate');
const existingUserCount = new Counter('existing_user_count');

export function setup() {
  const health = http.get(`${BASE_URL}/healthz`, requestOptions('healthz'));
  check(health, {
    'healthz is 200': (res) => res.status === 200,
  });

  return {
    startUserNumber: userOffset() + 1,
    endUserNumber: userOffset() + USER_COUNT,
  };
}

export default function () {
  const userNumber = userOffset() + exec.scenario.iterationInTest + 1;
  const user = userFromNumber(userNumber);

  const registerRes = http.post(`${BASE_URL}/auth/register`, JSON.stringify({
    name: user.name,
    email: user.email,
    password: user.password,
  }), {
    ...requestOptions('register_user'),
    headers: jsonHeaders,
  });

  if (registerRes.status === 201) {
    registerUserSuccessRate.add(true);
    check(registerRes, {
      'register user is 201': (res) => res.status === 201,
      'register user id exists': (res) => Boolean(safeJsonField(res, 'id')),
    });
    return;
  }

  const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
    email: user.email,
    password: user.password,
  }), {
    ...requestOptions('existing_user_login'),
    headers: jsonHeaders,
  });

  if (loginRes.status === 200) {
    existingUserCount.add(1);
    registerUserSuccessRate.add(true);
    check(loginRes, {
      'existing user can login': (res) => res.status === 200,
    });
    return;
  }

  registerUserSuccessRate.add(false);
  check(registerRes, {
    'register or login succeeds': () => false,
  });
}

export function teardown(data) {
  console.log(`Registered users are ready: ${data.startUserNumber}-${data.endUserNumber}`);
}

function requestOptions(name) {
  return {
    tags: { name },
  };
}

function safeJsonField(res, fieldName) {
  try {
    return res.json(fieldName);
  } catch {
    return '';
  }
}
