package postgres

var postgresSchema = []string{
	`CREATE TABLE client (
		id SERIAL PRIMARY KEY,
  		user_name TEXT NOT NULL,
		password TEXT NOT NULL,
		email TEXT,
		role TEXT,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		last_logged_in TIMESTAMP NOT NULL,
		UNIQUE(user_name)
	  );`,

	`CREATE TABLE exchange (
		id SERIAL PRIMARY KEY,
		exchange_name TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		UNIQUE(exchange_name)
	);`,

	`CREATE TABLE client_order_history (
		id SERIAL PRIMARY KEY,
		order_id TEXT NOT NULL,
		client_id INT NOT NULL,
		exchange_id INT NOT NULL,
		currency_pair TEXT NOT NULL,
		asset_type TEXT NOT NULL,
		order_type TEXT NOT NULL,
		amount DOUBLE PRECISION NOT NULL,
		rate DOUBLE PRECISION NOT NULL,
		fulfilled_on TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchange(id),
		FOREIGN KEY(client_id) REFERENCES client(id),
		UNIQUE(exchange_id, order_id)
	);`,

	`CREATE TABLE exchange_platform_trade_history (
		id SERIAL PRIMARY KEY,
		order_id TEXT NOT NULL,
		exchange_id INT NOT NULL,
		currency_pair VARCHAR(20) NOT NULL,
		asset_type TEXT NOT NULL,
		order_type TEXT DEFAULT 'NOT SPECIFIED' NOT NULL,
		amount DOUBLE PRECISION NOT NULL,
		rate DOUBLE PRECISION NOT NULL,
		fulfilled_on TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchange(id),
		UNIQUE(exchange_id, order_id)
	);`}
