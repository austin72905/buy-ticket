Feature: Ticket booking flow
  The frontend drives the booking flow by calling reservation, order, and payment APIs
  after the user explicitly confirms each step.

  Background:
    Given an event is on sale
    And the user has selected an event

  Scenario: User enters the queue before reserving tickets
    Given the sale status requires queueing
    When the frontend calls GET /sale/status with the selected event_id
    Then the backend returns can_join_queue as true
    When the user clicks "Join Queue"
    Then the frontend calls POST /queue/join
    And the backend returns a queue_token
    When the frontend polls GET /queue/status/{queueToken}
    Then the backend eventually returns status ready
    And the response includes a purchase_token

  Scenario: User reserves tickets after queue is ready
    Given the user has a valid purchase_token
    And the user has selected a section and quantity
    When the user clicks "Reserve Tickets"
    Then the frontend calls POST /reservations
    And the request includes user_id, event_id, section_id, quantity, hold_until, and purchase_token
    And the backend creates a reservation
    And the backend reduces the reservable quantity for the selected section

  Scenario: User confirms booking details before creating an order
    Given the backend has created a reservation
    When the frontend shows the confirmation page
    Then the user can review event, section, quantity, amount, and reservation expiry
    When the user clicks "Submit Order"
    Then the frontend calls POST /orders
    And the request includes reservation_id, order_no, and expires_at
    And the backend creates an order with status pending_payment

  Scenario: User returns to the seat selection step before creating an order
    Given the backend has created a reservation
    And the frontend is showing the confirmation page
    When the user clicks "Choose Again"
    Then the frontend calls POST /reservations/cancel
    And the request includes reservation_id and cancelled_at
    And the backend cancels the reservation
    And the backend releases the reserved quantity back to the section
    And the frontend navigates back to the selection page

  Scenario: User pays after the order is created
    Given the backend has created an order with status pending_payment
    When the frontend shows the payment page
    Then the user can enter payment information
    When the user clicks "Pay Now"
    Then the frontend calls POST /payments
    And the request includes order_id, payment_no, method, amount, and paid_at
    And the backend marks the payment as paid
    And the backend marks the order as paid
    And the backend confirms the reservation

  Scenario: Reservation cannot be created without a purchase token
    Given the user has not received a purchase_token
    When the frontend calls POST /reservations
    Then the backend rejects the request
    And the frontend keeps the user on the selection or queue page

  Scenario: Order is not created automatically after reservation
    Given the backend has created a reservation
    When the user has not clicked "Submit Order"
    Then the frontend must not call POST /orders
    And the backend must not create an order automatically

  Scenario: Payment is not created automatically after order creation
    Given the backend has created an order with status pending_payment
    When the user has not clicked "Pay Now"
    Then the frontend must not call POST /payments
    And the backend must not mark the order as paid automatically

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
