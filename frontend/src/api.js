const TOKEN_KEY = "token";

export const getToken = () => localStorage.getItem(TOKEN_KEY);
export const clearToken = () => localStorage.removeItem(TOKEN_KEY);

// Called when the gateway rejects our token (expired/invalid) so the app can show the login page.
let onUnauthorized = () => {};
export const setUnauthorizedHandler = (fn) => (onUnauthorized = fn);

async function request(path, options = {}) {
  const token = getToken();
  const res = await fetch(path, {
    ...options,
    headers: {
      "content-type": "application/json",
      ...(token && { authorization: `Bearer ${token}` }),
    },
  });
  const body = await res.json().catch(() => ({}));
  if (res.status === 401 && token) {
    clearToken();
    onUnauthorized();
  }
  if (!res.ok) throw new Error(body.error || `Request failed (${res.status})`);
  return body;
}

export async function login(username, password) {
  const body = await request("/auth/login", { method: "POST", body: JSON.stringify({ username, password }) });
  localStorage.setItem(TOKEN_KEY, body.token);
  return body.username;
}

export const getTickets = () => request("/tickets");
export const getBookings = () => request("/bookings");
export const bookTicket = (ticket_id, user_name) =>
  request("/bookings", { method: "POST", body: JSON.stringify({ ticket_id, user_name }) });
