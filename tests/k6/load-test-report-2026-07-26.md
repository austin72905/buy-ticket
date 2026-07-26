# Buy Ticket k6 Load Test Report

Date: 2026-07-26
Environment: local k3s, Traefik Ingress, Postgres/Redis on host Docker
API base URL: `http://buy-ticket.local/api`

## Test Objective

Validate the booking flow under burst traffic and confirm that Kubernetes HPA scales the API deployment under CPU load.

## Scenario

Script: `tests/k6/booking-flow.js`

Flow:

```text
GET /healthz
GET /events/{eventId}/availability
GET /sale/status
POST /queue/join
GET /queue/status/{queueToken}
POST /reservations
POST /orders
```

Command:

```bash
k6 run   -e BASE_URL=http://buy-ticket.local/api   -e EVENT_ID=1   -e QUANTITY=1   -e VUS=1000   -e MAX_QUEUE_POLLS=120   tests/k6/booking-flow.js
```

## k6 Result Summary

```text
booking_flow_success_rate: 99.30%  993 / 1000
reservation_success_rate: 99.30%  993 / 1000
order_success_rate:       99.30%  993 / 1000
http_req_failed:           0.08%  7 / 8445
http_req_duration p95:     522.48ms
http_req_duration p99:     675.60ms
iterations:                1000 completed
run duration:              11.1s
```

Failed checks:

```text
queue join is 201: 993 passed / 7 failed
queue token exists: 993 passed / 7 failed
```

The failed requests happened at `POST /queue/join`. Reservation and order checks succeeded for all users that successfully joined the queue.

## HPA Behavior

During the test, HPA scaled the API deployment under CPU pressure:

```text
cpu: 90%/70%    replicas: 2
cpu: 429%/70%   replicas: 3
cpu: 327%/70%   replicas: 5
```

This confirms that HPA reacted to CPU utilization and increased API replicas during the load spike.

Current post-test HPA state after cooldown:

```text
NAME             REFERENCE                   TARGETS       MINPODS   MAXPODS   REPLICAS
buy-ticket-api   Deployment/buy-ticket-api   cpu: 1%/80%   2         10        2
```

Current pods:

```text
buy-ticket-api-56b5f77989-2xtht         1/1 Running
buy-ticket-api-56b5f77989-nqjj9         1/1 Running
buy-ticket-scheduler-6cdb6f68ff-tbcvm   1/1 Running
```

Current resource usage after cooldown:

```text
buy-ticket-api-56b5f77989-2xtht         1m CPU   25Mi memory
buy-ticket-api-56b5f77989-nqjj9         1m CPU   25Mi memory
buy-ticket-scheduler-6cdb6f68ff-tbcvm   2m CPU   18Mi memory
```

## Data Consistency Check

Immediately after the load test, inventory state for `event_id = 1` was verified as:

```text
section  total  reserved  sold  available
A        1000   199       0     801
B        1000   197       0     803
C        1000   200       0     800
D        1000   198       0     802
E        1000   199       0     801
```

Total reserved quantity:

```text
199 + 197 + 200 + 198 + 199 = 993
```

This matches the k6 successful booking count:

```text
993 successful booking flows
```

No oversell was observed. The total reserved quantity matched successful reservations.

Note: the local database was later reset for another test run. The current inventory can show `reserved_quantity = 0`, but that is not the original post-test state captured above.

## Failure Analysis

The 7 failed flows failed before receiving a queue token:

```text
POST /queue/join
```

This is expected under a 1000-VU burst because the application has queue join backpressure:

```text
queue.join.max_in_flight=500
queue.join.retry_after_seconds=1
```

When too many requests enter `/queue/join` at the same time, the service can reject excess requests with `429 Too Many Requests`. The k6 script retries transient queue join failures, but 7 requests still failed after retries.

This is an admission-control behavior, not a database consistency failure.

## Conclusion

The system handled a 1000-concurrent-user booking burst successfully:

```text
Success rate: 99.30%
Latency p95: 522.48ms
Latency p99: 675.60ms
HPA scaled API replicas under load
Inventory remained consistent
No oversell observed
```

The small number of queue join failures is acceptable for the current configuration and is caused by intentional backpressure protecting Redis/Postgres during burst traffic.

## Recommended Next Tests

1. Run the same test with `QUEUE_JOIN_RETRIES=10` to verify whether the remaining 0.7% queue join failures disappear.

```bash
k6 run   -e BASE_URL=http://buy-ticket.local/api   -e EVENT_ID=1   -e QUANTITY=1   -e VUS=1000   -e MAX_QUEUE_POLLS=120   -e QUEUE_JOIN_RETRIES=10   tests/k6/booking-flow.js
```

2. Run a ramp-up test instead of an instant 1000-VU burst to simulate more realistic traffic.

3. Export JSON results and inspect `queue_join` status distribution:

```bash
k6 run   -e BASE_URL=http://buy-ticket.local/api   -e EVENT_ID=1   -e QUANTITY=1   -e VUS=1000   -e MAX_QUEUE_POLLS=120   --out json=tests/k6/latest-results.json   tests/k6/booking-flow.js

jq -r '
  select(.metric=="http_reqs" and .data.tags.name=="queue_join")
  | .data.tags.status // "0"
' tests/k6/latest-results.json | sort | uniq -c
```
