const { Pool } = require("pg");

const pool = new Pool({ connectionString: process.env.DATABASE_URL });

async function init() {
  await pool.query(`
    CREATE TABLE IF NOT EXISTS tickets (
      id SERIAL PRIMARY KEY,
      event TEXT NOT NULL,
      price NUMERIC(10, 2) NOT NULL,
      status TEXT NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'booked'))
    )
  `);
  const { rows } = await pool.query("SELECT COUNT(*)::int AS n FROM tickets");
  if (rows[0].n === 0) {
    await pool.query(`
      INSERT INTO tickets (event, price) VALUES
        ('Coldplay - Dhaka', 120.00),
        ('Coldplay - Dhaka', 120.00),
        ('Jazz Night', 45.50),
        ('Jazz Night', 45.50),
        ('Tech Conf 2026', 200.00)
    `);
  }
}

module.exports = { pool, init };
