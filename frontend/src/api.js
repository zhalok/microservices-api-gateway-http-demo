async function request(path, options) {
  const res = await fetch(path, {
    headers: { "content-type": "application/json" },
    ...options,
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(body.error || `Request failed (${res.status})`);
  return body;
}

export const getTickets = () => request("/tickets");
export const getBookings = () => request("/bookings");
export const bookTicket = (ticket_id, user_name) =>
  request("/bookings", { method: "POST", body: JSON.stringify({ ticket_id, user_name }) });
