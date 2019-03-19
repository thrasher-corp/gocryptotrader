package postgres

var postgresSchema = []string{
	`CREATE TABLE users (
		id SERIAL PRIMARY KEY,
  		user_name TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		email TEXT UNIQUE,
		one_time_password TEXT,
		password_created_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		last_logged_in TIMESTAMP NOT NULL,
		enabled BOOLEAN NOT NULL
	  );`,

	`CREATE TABLE exchanges (
		id SERIAL PRIMARY KEY,
		exchange_name TEXT NOT NULL UNIQUE,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);`,

	`CREATE TABLE keys (
		id SERIAL PRIMARY KEY,
		api_key TEXT NOT NULL,
		api_secret TEXT NOT NULL,
		exchange_id INT NOT NULL,
		expires_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		enabled BOOLEAN NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchanges(id)
	);`,

	`CREATE TABLE user_keys (
		id SERIAL PRIMARY KEY,
		key_id INT NOT NULL,
		user_id INT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(key_id) REFERENCES keys(id)
	);`,

	`CREATE TABLE audit_trails (
		id BIGSERIAL PRIMARY KEY,
		user_id INT NOT NULL,
		change TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`,

	`CREATE TABLE user_order_history (
		id BIGSERIAL PRIMARY KEY,
		order_id TEXT NOT NULL,
		user_id INT NOT NULL,
		exchange_id INT NOT NULL,
		currency_pair TEXT NOT NULL,
		asset_type TEXT NOT NULL,
		order_type TEXT NOT NULL,
		amount DOUBLE PRECISION NOT NULL,
		rate DOUBLE PRECISION NOT NULL,
		fulfilled_on TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchanges(id),
		FOREIGN KEY(user_id) REFERENCES users(id),
		UNIQUE(exchange_id, order_id)
	);`,

	`CREATE TABLE exchange_platform_trade_history (
		id BIGSERIAL PRIMARY KEY,
		order_id TEXT NOT NULL,
		exchange_id INT NOT NULL,
		currency_pair VARCHAR(20) NOT NULL,
		asset_type TEXT NOT NULL,
		order_type TEXT DEFAULT 'NOT SPECIFIED' NOT NULL,
		amount DOUBLE PRECISION NOT NULL,
		rate DOUBLE PRECISION NOT NULL,
		fulfilled_on TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(exchange_id) REFERENCES exchanges(id),
		UNIQUE(exchange_id, order_id)
	);`,
}
