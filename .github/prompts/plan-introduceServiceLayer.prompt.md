# Plan: Introduce Service Layer Between Handlers and Repositories

Extract business logic from handlers ([checkout.go](internal/handlers/checkout.go), [session.go](internal/handlers/session.go), [confirmation.go](internal/handlers/confirmation.go)) into dedicated service layer components. The service layer will contain order management, payment orchestration, and Adyen API integration logic, leaving handlers focused solely on HTTP request/response handling.

## Steps

1. **Create service layer structure**: Add `internal/services/` directory with `order_service.go`, `payment_service.go`, and `adyen_client.go` files with interface definitions and constructor functions.

2. **Extract Adyen API client**: Move HTTP communication logic from [session.go](internal/handlers/session.go#L145-L190) and [confirmation.go](internal/handlers/confirmation.go#L143-L206) into `AdyenClient` with `CreateSession()` and `GetSessionStatus()` methods, handling environment-based endpoint selection and API key injection.

3. **Build PaymentService**: Extract payment session creation orchestration from [session.go](internal/handlers/session.go#L95-L220) and payment verification from [confirmation.go](internal/handlers/confirmation.go#L82-L220), including order creation, Adyen integration, and status mapping business rules (like `mapResultCodeToStatus()`).

4. **Build OrderService**: Extract order operations into service methods - create order with UUID/reference generation, retrieve order, and update order status - wrapping calls to [order_repo.go](internal/repository/order_repo.go) with business validation.

5. **Create unit tests for services**: Write unit tests for `OrderService`, `PaymentService`, and `AdyenClient` using mocked dependencies (mock repository, mock HTTP client) to validate business logic in isolation before integration.

6. **Update dependency injection**: Modify [main.go](cmd/simplecom/main.go) `buildServerDependencies()` to instantiate services (injecting repository and config), then inject services into handlers instead of repository directly.

7. **Refactor handlers to thin layer**: Update [session.go](internal/handlers/session.go) and [confirmation.go](internal/handlers/confirmation.go) to only handle HTTP concerns - parse requests, call service methods, handle service errors, and render responses - removing all business logic.

8. **Verify with E2E tests**: Run existing E2E tests to ensure refactoring maintains all functionality.

## Deferred for Later

1. **Transaction support**: Database transaction handling for atomic multi-step operations will be added in a future iteration.

2. **Error handling strategy**: Migration from string-based errors to typed error structs will be addressed separately.
