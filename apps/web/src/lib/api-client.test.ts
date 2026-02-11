import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("api client critical request flows", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.restoreAllMocks();
    process.env.NEXT_PUBLIC_API_BASE_URL = "https://api.example.com/api/v1";
    delete process.env.MARKETPLACE_API_BASE_URL;
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("rejects checkout place-order payload without a valid idempotency key", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    const { placeOrder } = await import("./api-client");

    await expect(placeOrder({ idempotency_key: "x" }, "gst_invalid")).rejects.toThrow();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("sends stripe intent request with validated payload and guest token header", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          id: "pay_1",
          order_id: "ord_1",
          method: "stripe",
          status: "pending",
          provider: "stripe",
          provider_ref: "pi_1",
          client_secret: "secret_1",
          amount_cents: 1000,
          currency: "USD",
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        }),
        {
          status: 201,
          headers: {
            "Content-Type": "application/json",
            "X-Guest-Token": "gst_from_header",
          },
        },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const { createStripePaymentIntent } = await import("./api-client");

    const result = await createStripePaymentIntent(
      { order_id: "ord_1", idempotency_key: "pi_12345678" },
      "gst_client",
    );

    expect(fetchMock).toHaveBeenCalledTimes(1);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("https://api.example.com/api/v1/payments/stripe/intent");
    expect(init.method).toBe("POST");

    const headers = init.headers as Headers;
    expect(headers.get("X-Guest-Token")).toBe("gst_client");
    expect(headers.get("Content-Type")).toBe("application/json");
    expect(JSON.parse(String(init.body))).toEqual({
      order_id: "ord_1",
      idempotency_key: "pi_12345678",
    });

    expect(result.payload.order_id).toBe("ord_1");
    expect(result.guestToken).toBe("gst_from_header");
  });
});
