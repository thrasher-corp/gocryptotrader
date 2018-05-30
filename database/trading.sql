CREATE TABLE config (
  config_id bigint NOT NULL,
  name text NOT NULL,
  password text NOT NULL,
  encrypt_config int NOT NULL,
  cryptocurrencies text NOT NULL,
  currency_fx_provider text NOT NULL,
  fiat_display_currency text NOT NULL,
  global_http_timeout bigint NOT NULL
);

ALTER TABLE config ADD CONSTRAINT config_pkey PRIMARY KEY (config_id);

CREATE TABLE portfolio (
  portfolio_id bigint NOT NULL,
  config_id bigint NOT NULL,
  coin_address text NOT NULL,
  coin_type text NOT NULL,
  balance double precision NOT NULL,
  description text NOT NULL
);

ALTER TABLE portfolio ADD CONSTRAINT portfolio_pkey PRIMARY KEY (portfolio_id);

CREATE TABLE taxable_events (
  taxable_events_id bigint NOT NULL,
  config_id bigint NOT NULL,
  conversion_from text NOT NULL,
  conversion_from_amount double precision NOT NULL,
  conversion_from_amount_equivalant_value double precision NOT NULL,
  conversion_to text NOT NULL,
  conversion_to_amount double precision NOT NULL,
  conversion_toamount_equivalant_value double precision NOT NULL,
  conversion_gain_loss double precision NOT NULL,
  exchange_name text NOT NULL,
  date_and_time timestamptz NOT NULL
);

ALTER TABLE taxable_events ADD CONSTRAINT taxable_events_pkey PRIMARY KEY (taxable_events_id);

CREATE TABLE smsglobal (
  smsglobal_id bigint NOT NULL,
  config_id bigint NOT NULL,
  enabled boolean NOT NULL,
  username text NOT NULL,
  password text NOT NULL
);

ALTER TABLE smsglobal ADD CONSTRAINT smsglobal_pkey PRIMARY KEY (smsglobal_id);

CREATE TABLE smsglobal_contacts (
  smsglobal_contacts_id bigint NOT NULL,
  config_id bigint NOT NULL,
  name text NOT NULL,
  phone_number text NOT NULL,
  enabled boolean NOT NULL
);

ALTER TABLE smsglobal_contacts ADD CONSTRAINT smsglobal_contacts_pkey PRIMARY KEY (smsglobal_contacts_id);

CREATE TABLE webserver (
  webserver_id bigint NOT NULL,
  config_id bigint NOT NULL,
  enabled boolean NOT NULL,
  admin_username text NOT NULL,
  admin_password text NOT NULL,
  listen_address text NOT NULL,
  websocket_connection_limit integer NOT NULL,
  websocket_allow_insecure_origin boolean NOT NULL
);

ALTER TABLE webserver ADD CONSTRAINT webserver_pkey PRIMARY KEY (webserver_id);

CREATE TABLE exchanges (
  exchange_id bigint NOT NULL,
  config_id bigint NOT NULL,
  exchange_name text NOT NULL,
  enabled boolean NOT NULL,
  is_verbose boolean NOT NULL,
  websocket boolean NOT NULL,
  use_sandbox boolean NOT NULL,
  rest_polling_delay bigint NOT NULL,
  http_timeout bigint NOT NULL,
  authenticated_api_support boolean NOT NULL,
  api_key text,
  api_secret text,
  client_id text,
  available_pairs text NOT NULL,
  enabled_pairs text NOT NULL,
  base_currencies text NOT NULL,
  asset_types text,
  supported_auto_pair_updates boolean NOT NULL,
  pairs_last_updated timestamptz NOT NULL
);

ALTER TABLE exchanges ADD CONSTRAINT exchange_pkey PRIMARY KEY (exchange_id);

CREATE TABLE currency_pair_format (
  currency_pair_format_id bigint NOT NULL,
  exchange_id bigint NOT NULL,
  config_currency boolean NOT NULL,
  request_currency boolean NOT NULL,
  name text NOT NULL,
  uppercase boolean NOT NULL,
  delimiter text NOT NULL,
  separator text NOT NULL,
  index text NOT NULL
);

ALTER TABLE currency_pair_format ADD CONSTRAINT currency_pair_format_pkey PRIMARY KEY (currency_pair_format_id);
ALTER TABLE currency_pair_format ADD CONSTRAINT exchange_currency_pair_format_fkey FOREIGN KEY (exchange_id) REFERENCES exchanges(exchange_id);


CREATE TABLE order_history (
  order_history_id bigint NOT NULL,
  config_id bigint NOT NULL,
  exchange_id text NOT NULL,
  transaction_id bigint NOT NULL,
  fulfilled_on timestamp NOT NULL,
  currency_pair text NOT NULL,
  asset_type text NOT NULL,
  order_type text NOT NULL,
  amount double precision NOT NULL,
  rate double precision NOT NULL
);

ALTER TABLE order_history ADD CONSTRAINT order_history_pkey PRIMARY KEY (order_history_id);
ALTER TABLE order_history ADD CONSTRAINT config_order_history_fkey FOREIGN KEY (config_id) REFERENCES config(config_id);

CREATE TABLE exchange_trade_history (
  exchange_trade_history_id bigint NOT NULL,
  config_id bigint NOT NULL,
  exchange_id bigint NOT NULL,
  fulfilled_on timestamp NOT NULL,
  currency_pair text NOT NULL,
  asset_type text NOT NULL,
  order_type text NOT NULL,
  amount double precision NOT NULL,
  rate double precision NOT NULL
);

ALTER TABLE exchange_trade_history ADD CONSTRAINT exchange_trade_history_pkey PRIMARY KEY (exchange_trade_history_id);
