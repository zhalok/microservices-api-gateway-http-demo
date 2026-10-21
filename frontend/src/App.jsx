import { useCallback, useEffect, useState } from "react";
import { bookTicket, clearToken, getBookings, getTickets, getToken, setUnauthorizedHandler } from "./api.js";
import Login from "./Login.jsx";

// The JWT's `sub` claim is the username; decode it so a page reload keeps the session.
function usernameFromToken(token) {
  try {
    const { sub, exp } = JSON.parse(atob(token.split(".")[1]));
    return exp * 1000 > Date.now() ? sub : null;
  } catch {
    return null;
  }
}

export default function App() {
  const [user, setUser] = useState(() => {
    const token = getToken();
    return token ? usernameFromToken(token) : null;
  });

  useEffect(() => setUnauthorizedHandler(() => setUser(null)), []);

  if (!user) return <Login onLogin={setUser} />;
  return (
    <Tickets
      user={user}
      onLogout={() => {
        clearToken();
        setUser(null);
      }}
    />
  );
}

function Tickets({ user, onLogout }) {
  const [tickets, setTickets] = useState([]);
  const [bookings, setBookings] = useState([]);
  const [message, setMessage] = useState(null); // { type: "ok" | "error", text }
  const [busyId, setBusyId] = useState(null);

  const load = useCallback(async () => {
    try {
      const [t, b] = await Promise.all([getTickets(), getBookings()]);
      setTickets(t);
      setBookings(b);
    } catch (e) {
      setMessage({ type: "error", text: e.message });
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function handleBook(ticket) {
    setBusyId(ticket.id);
    try {
      await bookTicket(ticket.id, user);
      setMessage({ type: "ok", text: `Booked ticket #${ticket.id} for ${user}.` });
    } catch (e) {
      setMessage({ type: "error", text: e.message });
    } finally {
      setBusyId(null);
      load(); // refresh either way: a 409 means someone else got it
    }
  }

  const eventOf = (ticketId) => tickets.find((t) => t.id === ticketId)?.event ?? "—";

  // group tickets by event
  const events = tickets.reduce((acc, t) => {
    (acc[t.event] ||= []).push(t);
    return acc;
  }, {});

  return (
    <main>
      <h1>🎟️ Ticket Booking</h1>

      <div className="name-row">
        Signed in as <strong>{user}</strong>{" "}
        <button onClick={onLogout}>Log out</button>
      </div>

      {message && <p className={`message ${message.type}`}>{message.text}</p>}

      <h2>Events</h2>
      {Object.entries(events).map(([event, list]) => {
        const available = list.filter((t) => t.status === "available").length;
        return (
          <section key={event} className="event">
            <header>
              <h3>{event}</h3>
              <span>
                ${list[0].price} · {available}/{list.length} available
              </span>
            </header>
            <div className="tickets">
              {list.map((t) => (
                <button
                  key={t.id}
                  className={`ticket ${t.status}`}
                  disabled={t.status !== "available" || busyId === t.id}
                  onClick={() => handleBook(t)}
                  title={t.status === "available" ? "Click to book" : "Already booked"}
                >
                  #{t.id}
                </button>
              ))}
            </div>
          </section>
        );
      })}

      <h2>Bookings</h2>
      {bookings.length === 0 ? (
        <p className="muted">No bookings yet.</p>
      ) : (
        <table>
          <thead>
            <tr><th>ID</th><th>Event</th><th>Ticket</th><th>Name</th><th>Time</th></tr>
          </thead>
          <tbody>
            {bookings.map((b) => (
              <tr key={b.id}>
                <td>{b.id}</td>
                <td>{eventOf(b.ticket_id)}</td>
                <td>#{b.ticket_id}</td>
                <td>{b.user_name}</td>
                <td>{new Date(b.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </main>
  );
}
