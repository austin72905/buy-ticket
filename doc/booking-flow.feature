Feature: Ticket booking flow
  The frontend drives the booking flow by calling auth, queue, reservation,
  order, and payment APIs after the user explicitly confirms each step.

  Background:
    Given an event is on sale
    And the user has registered or logged in
    And the backend has set the buy_ticket_session cookie
    And the user has selected an event

  Scenario: User enters the queue before reserving tickets
    Given the sale status requires queueing
    When the frontend calls GET /sale/status with the selected event_id
    Then the backend returns can_join_queue as true
    When the user clicks "Join Queue"
    Then the frontend calls POST /queue/join
    And the request includes event_id, client_id, request_id, and channel
    And the request does not include user_id
    And the backend reads the user from the session
    And the backend returns a queue_token
    When the frontend polls GET /queue/status/{queueToken}
    Then the backend eventually returns status ready
    And the response includes a purchase_token
    And the purchase_token remains valid until it expires or an order is created

  Scenario: User reserves tickets after queue is ready
    Given the user has a valid purchase_token
    And the user has selected a section and quantity
    When the user clicks "Reserve Tickets"
    Then the frontend calls POST /reservations
    And the request includes event_id, section_id, quantity, hold_until, and purchase_token
    And the request does not include user_id
    And the backend validates the purchase_token
    And the backend creates a reservation with status holding
    And the backend reduces the reservable quantity for the selected section
    And the backend does not consume the purchase_token yet

  Scenario: User cannot create two active reservations for the same event
    Given the user has a valid purchase_token
    And the backend has created an active reservation for the selected event
    When the frontend calls POST /reservations again without cancelling the active reservation
    Then the backend rejects the request
    And the backend does not create another active reservation
    And the backend does not consume the purchase_token

  Scenario: User returns to the seat selection step before creating an order
    Given the backend has created a reservation with status holding
    And the frontend is showing the confirmation page
    When the user clicks "Choose Again"
    Then the frontend calls POST /reservations/cancel
    And the request includes reservation_id and cancelled_at
    And the backend cancels the reservation
    And the backend releases the reserved quantity back to the section
    And the purchase_token remains usable while it is not expired
    And the frontend navigates back to the selection page

  Scenario: User creates a different reservation after choosing again
    Given the user has cancelled the previous reservation before creating an order
    And the original purchase_token is still valid
    And the user has selected a different section or quantity
    When the user clicks "Reserve Tickets"
    Then the frontend calls POST /reservations
    And the request includes the original purchase_token
    And the backend creates a new reservation with status holding

  Scenario: User confirms booking details before creating an order
    Given the backend has created a reservation with status holding
    And the user has a valid purchase_token
    When the frontend shows the confirmation page
    Then the user can review event, section, quantity, amount, and reservation expiry
    When the user clicks "Submit Order"
    Then the frontend calls POST /orders
    And the request includes reservation_id, order_no, expires_at, and purchase_token
    And the backend validates and consumes the purchase_token
    And the backend creates an order with status pending_payment
    And the user can no longer use the same purchase_token to create another reservation

  Scenario: User pays after the order is created
    Given the backend has created an order with status pending_payment
    When the frontend shows the payment page
    Then the user can review the payment amount and method
    When the user clicks "Pay Now"
    Then the frontend calls POST /payments/start
    And the request includes order_id, method, and provider
    And the request includes an Idempotency-Key header
    And the backend loads the order
    And the backend sets amount from order.total_amount
    And the backend creates a payment_attempt
    And the backend sends the attempt to the mock payment service
    And the backend waits for provider callback before marking the order as paid

  Scenario: Mock payment callback completes payment
    Given the frontend has started a payment attempt
    And the mock payment service has accepted the attempt
    When the mock payment service calls POST /payments/provider/ecpay/callback
    Then the backend finds the payment_attempt by MerchantTradeNo
    And the backend generates payment_no
    And the backend sets paid_at from server time
    And the backend marks the payment_attempt as succeeded
    And the backend marks the payment as paid
    And the backend marks the order as paid
    And the backend confirms the reservation

  Scenario: Reservation cannot be created without a purchase token
    Given the user has not received a purchase_token
    When the frontend calls POST /reservations
    Then the backend rejects the request
    And the frontend keeps the user on the selection or queue page

  Scenario: Order cannot be created without a purchase token
    Given the backend has created a reservation with status holding
    When the frontend calls POST /orders without purchase_token
    Then the backend rejects the request
    And the backend does not create an order

  Scenario: Order is not created automatically after reservation
    Given the backend has created a reservation
    When the user has not clicked "Submit Order"
    Then the frontend must not call POST /orders
    And the backend must not create an order automatically
    And the purchase_token remains unconsumed

  Scenario: Payment is not created automatically after order creation
    Given the backend has created an order with status pending_payment
    When the user has not clicked "Pay Now"
    Then the frontend must not call POST /payments/start
    And the backend must not mark the order as paid automatically

  Scenario: Expired purchase tokens are swept by a background job
    Given the user has a ready queue status with a purchase_token
    And the purchase_token expires before an order is created
    When the purchase token cleanup scheduler runs
    Then the backend clears the purchase_token
    And the user cannot create a reservation or order with that purchase_token

  Scenario: Expired pending orders are swept by a background job
    Given the backend has created an order with status pending_payment
    And the order expires before payment succeeds
    When the order expire scheduler runs
    Then the backend marks the order as expired
    And the backend expires the related reservation
    And the backend releases the reserved quantity back to the section

  Scenario: Queue status is advanced by a background job
    Given users are waiting in the queue
    When the queue promotion scheduler runs
    Then waiting users may be promoted to ready
    And promoted users receive a purchase_token
