import { cookies } from "next/headers";

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  process.env.MARKETPLACE_API_BASE_URL ??
  "http://localhost:8080/api/v1";
const guestTokenHeader = "X-Guest-Token";

interface RouteContext {
  params: Promise<{ orderID: string }>;
}

export async function GET(_request: Request, context: RouteContext): Promise<Response> {
  const { orderID } = await context.params;
  const normalizedOrderID = String(orderID ?? "").trim();
  if (!normalizedOrderID) {
    return Response.json({ error: "order id is required" }, { status: 400 });
  }

  const headers = new Headers();
  const guestToken = (await cookies()).get("mkt_guest_token")?.value;
  if (guestToken) {
    headers.set(guestTokenHeader, guestToken);
  }

  const upstream = await fetch(`${API_BASE_URL}/invoices/${encodeURIComponent(normalizedOrderID)}/download`, {
    method: "GET",
    headers,
    cache: "no-store",
  });

  if (!upstream.ok) {
    return Response.json({ error: "invoice download failed" }, { status: upstream.status });
  }

  const contentDisposition =
    upstream.headers.get("Content-Disposition") ?? `attachment; filename=invoice-${normalizedOrderID}.pdf`;
  const payload = await upstream.arrayBuffer();

  return new Response(payload, {
    status: 200,
    headers: {
      "Content-Type": "application/pdf",
      "Content-Disposition": contentDisposition,
      "Cache-Control": "no-store",
    },
  });
}
