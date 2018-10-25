package database

var databaseTables = map[string]string{
	"config": `CREATE TABLE config (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		config_name text NOT NULL,
		global_http_timeout integer NOT NULL,
		webserver_enabled boolean NOT NULL DEFAULT false,
		webserver_admin_username text,
		webserver_admin_password text,
		webserver_listen_address text,
		webserver_websocket_connection_limit integer,
		webserver_allow_insecure_origin boolean NOT NULL DEFAULT false
	  );`,

	"portfolio": `CREATE TABLE portfolio (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	config_id integer NOT NULL,
	FOREIGN KEY(config_id) REFERENCES config(id)
  );`,

	"cryptocurrency_portfolio_address": `CREATE TABLE cryptocurrency_portfolio_address (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	coin_name text NOT NULL,
	coin_address text NOT NULL,
	is_hot_wallet boolean NOT NULL DEFAULT false,
	portfolio_id integer NOT NULL,
	FOREIGN KEY(portfolio_id) REFERENCES portfolio(id)
  );`,

	"foreign_exchange_provider_config": `CREATE TABLE foreign_exchange_provider_config (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	name text NOT NULL,
	enabled boolean NOT NULL DEFAULT false,
	verbose boolean NOT NULL DEFAULT false,
	rest_polling_delay integer NOT NULL,
	api_key text,
	api_key_level integer,
	is_primary_provider boolean NOT NULL DEFAULT false,
	config_id integer NOT NULL,
	FOREIGN KEY(config_id) REFERENCES config(id)
  );`,

	"communication_config": `CREATE TABLE communication_config (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	name text NOT NULL,
	enabled boolean NOT NULL DEFAULT false,
	verbose boolean NOT NULL DEFAULT false,
	account_username text,
	account_password text,
	target_channel text,
	verification_token text,
	host text,
	port text,
	config_id integer NOT NULL,
	FOREIGN KEY(config_id) REFERENCES config(id)
  );`,

	"communication_config_contact": `CREATE TABLE communication_config_contact (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	name text,
	email text,
	phone_number text,
	enabled boolean NOT NULL DEFAULT false,
	config_id integer NOT NULL,
	FOREIGN KEY(config_id) REFERENCES config(id)
  );`,

	"exchange_config": `CREATE TABLE exchange_config (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	name text NOT NULL,
	enabled boolean NOT NULL DEFAULT false,
	verbose boolean NOT NULL DEFAULT false,
	websocket_enabled boolean NOT NULL DEFAULT false,
	use_sandbox boolean NOT NULL DEFAULT false,
	rest_polling_delay integer NOT NULL,
	http_timeout integer NOT NULL,
	authenticated_api_support boolean NOT NULL DEFAULT false,
	api_key text,
	api_secret text,
	client_id text,
	available_pairs text NOT NULL,
	enabled_pairs text NOT NULL,
	base_currencies text NOT NULL,
	asset_types text NOT NULL,
	supported_auto_pair_updates boolean NOT NULL DEFAULT false,
	pairs_last_updated DATETIME NOT NULL,
	config_id integer NOT NULL,
	FOREIGN KEY(config_id) REFERENCES config(id)
  );`,

	"exchange_currency_pair_format": `CREATE TABLE exchange_currency_pair_format (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	name text NOT NULL,
	uppercase integer NOT NULL,
	delimiter text NOT NULL,
	separator text NOT NULL,
	index_name text NOT NULL,
	exchange_id integer NOT NULL,
	FOREIGN KEY(exchange_id) REFERENCES exchange_config(id)
  );`,

	"exchange_portfolio_order_history": `CREATE TABLE exchange_portfolio_order_history (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	fulfilled_on DATETIME NOT NULL,
	currency_pair text NOT NULL,
	asset_type text NOT NULL,
	order_type text NOT NULL,
	contract_type text NOT NULL,
	amount real NOT NULL,
	rate real NOT NULL,
	exchange_config_id integer NOT NULL,
	portfolio_id integer NOT NULL,
	FOREIGN KEY(portfolio_id) REFERENCES portfolio(id)
  );`,

	"exchange_trade_history": `CREATE TABLE exchange_trade_history (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	fulfilled_on DATETIME NOT NULL,
	currency_pair text NOT NULL,
	asset_type text NOT NULL,
	order_type text NOT NULL DEFAULT "NOT SPECIFIED",
	contract_type text NOT NULL,
	amount real NOT NULL,
	rate real NOT NULL,
	order_id integer,
	exchange_id integer NOT NULL,
	FOREIGN KEY(exchange_id) REFERENCES exchange_config(id)
  );`,

	"bank_account_config": `CREATE TABLE bank_account_config (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	bank_name text NOT NULL,
	bank_address text NOT NULL,
	account_name text NOT NULL,
	account_number text NOT NULL,
	is_exchange_bank boolean NOT NULL DEFAULT false,
	swift_code text,
	iban text,
	bsb_number text,
	supported_currencies text NOT NULL,
	supported_exchanges text,
	config_id integer NOT NULL,
	FOREIGN KEY(config_id) REFERENCES config(id)
  );`}

var deprecatedDatabaseTables = []string{}

var fullSchema = `CREATE TABLE config (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  config_name text NOT NULL,
  global_http_timeout integer NOT NULL,
  webserver_enabled boolean NOT NULL DEFAULT false,
  webserver_admin_username text,
  webserver_admin_password text,
  webserver_listen_address text,
  webserver_websocket_connection_limit integer,
  webserver_allow_insecure_origin boolean NOT NULL DEFAULT false
);

CREATE TABLE portfolio (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  config_id integer NOT NULL,
  FOREIGN KEY(config_id) REFERENCES config(id)
);

CREATE TABLE cryptocurrency_portfolio_address (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  coin_name text NOT NULL,
  coin_address text NOT NULL,
  is_hot_wallet boolean NOT NULL DEFAULT false,
  portfolio_id integer NOT NULL,
  FOREIGN KEY(portfolio_id) REFERENCES portfolio(id)
);

CREATE TABLE foreign_exchange_provider_config (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  name text NOT NULL,
  enabled boolean NOT NULL DEFAULT false,
  verbose boolean NOT NULL DEFAULT false,
  rest_polling_delay integer NOT NULL,
  api_key text,
  api_key_level integer,
  is_primary_provider boolean NOT NULL DEFAULT false,
  config_id integer NOT NULL,
  FOREIGN KEY(config_id) REFERENCES config(id)
);

CREATE TABLE communication_config (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  name text NOT NULL,
  enabled boolean NOT NULL DEFAULT false,
  verbose boolean NOT NULL DEFAULT false,
  account_username text,
  account_password text,
  target_channel text,
  verification_token text,
  host text,
  port text,
  config_id integer NOT NULL,
  FOREIGN KEY(config_id) REFERENCES config(id)
);

CREATE TABLE communication_config_contact (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  name text,
  email text,
  phone_number text,
  enabled boolean NOT NULL DEFAULT false,
  config_id integer NOT NULL,
  FOREIGN KEY(config_id) REFERENCES config(id)
);


CREATE TABLE exchange_config (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  name text NOT NULL,
  enabled boolean NOT NULL DEFAULT false,
  verbose boolean NOT NULL DEFAULT false,
  websocket_enabled boolean NOT NULL DEFAULT false,
  use_sandbox boolean NOT NULL DEFAULT false,
  rest_polling_delay integer NOT NULL,
  http_timeout integer NOT NULL,
  authenticated_api_support boolean NOT NULL DEFAULT false,
  api_key text,
  api_secret text,
  client_id text,
  available_pairs text NOT NULL,
  enabled_pairs text NOT NULL,
  base_currencies text NOT NULL,
  asset_types text NOT NULL,
  supported_auto_pair_updates boolean NOT NULL DEFAULT false,
  pairs_last_updated DATETIME NOT NULL,
  config_id integer NOT NULL,
  FOREIGN KEY(config_id) REFERENCES config(id)
);

CREATE TABLE exchange_currency_pair_format (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  name text NOT NULL,
  uppercase integer NOT NULL,
  delimiter text NOT NULL,
  separator text NOT NULL,
  index_name text NOT NULL,
  exchange_id integer NOT NULL,
  FOREIGN KEY(exchange_id) REFERENCES exchange_config(id)
);

CREATE TABLE exchange_portfolio_order_history (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  fulfilled_on DATETIME NOT NULL,
  currency_pair text NOT NULL,
  asset_type text NOT NULL,
  order_type text NOT NULL,
  contract_type text NOT NULL,
  amount real NOT NULL,
  rate real NOT NULL,
  exchange_config_id integer NOT NULL,
  portfolio_id integer NOT NULL,
  FOREIGN KEY(portfolio_id) REFERENCES portfolio(id)
);

CREATE TABLE exchange_trade_history (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  fulfilled_on DATETIME NOT NULL,
  currency_pair text NOT NULL,
  asset_type text NOT NULL,
  order_type text NOT NULL DEFAULT "NOT SPECIFIED",
  contract_type text NOT NULL,
  amount real NOT NULL,
  rate real NOT NULL,
  order_id integer,
  exchange_id integer NOT NULL,
  FOREIGN KEY(exchange_id) REFERENCES exchange_config(id)
);

CREATE TABLE bank_account_config (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  bank_name text NOT NULL,
  bank_address text NOT NULL,
  account_name text NOT NULL,
  account_number text NOT NULL,
  is_exchange_bank boolean NOT NULL DEFAULT false,
  swift_code text,
  iban text,
  bsb_number text,
  supported_currencies text NOT NULL,
  supported_exchanges text,
  config_id integer NOT NULL,
  FOREIGN KEY(config_id) REFERENCES config(id)
);`
