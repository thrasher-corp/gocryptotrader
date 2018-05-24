CREATE TABLE gct_user (
  id integer NOT NULL,
  name text NOT NULL,
  password text NOT NULL
);

ALTER TABLE gct_user ADD CONSTRAINT gct_user_pkey PRIMARY KEY (id);

CREATE TABLE exchange (
  id integer NOT NULL,
  gct_user_id integer NOT NULL,
  name text NOT NULL,
  enabled boolean NOT NULL,
  api_key text,
  api_secret text
);

ALTER TABLE exchange ADD CONSTRAINT exchange_pkey PRIMARY KEY (id);
ALTER TABLE exchange ADD CONSTRAINT exchange_users_fkey FOREIGN KEY (gct_user_id) REFERENCES gct_user(id);

CREATE TABLE ticker (
  id integer NOT NULL,
  exchange_id integer NOT NULL,
  executed_on timestamp NOT NULL,
  open real NOT NULL,
  high real NOT NULL,
  low real NOT NULL,
  close real NOT NULL,
  volume real NOT NULL,
  adj_close real NOT NULL
);

ALTER TABLE ticker ADD CONSTRAINT ticker_pkey PRIMARY KEY (id);
ALTER TABLE ticker ADD CONSTRAINT exchange_ticker_fkey FOREIGN KEY (exchange_id) REFERENCES exchange(id);

CREATE TABLE address_information (
  id integer NOT NULL,
  address text NOT NULL,
  coin_type text NOT NULL,
  balance real NOT NULL,
  description text
);

ALTER TABLE address_information ADD CONSTRAINT address_information_pkey PRIMARY KEY (id);

CREATE TABLE trade_History (
  gct_user_id integer NOT NULL,
  exchange_name text NOT NULL,
  fulfilled_on timestamp NOT NULL,
  currency_pair text NOT NULL,
  order_type text NOT NULL,
  amount real NOT NULL,
  rate real NOT NULL
);

ALTER TABLE trade_History ADD CONSTRAINT trade_History_pkey PRIMARY KEY (fulfilled_on);
ALTER TABLE trade_History ADD CONSTRAINT gct_user_trade_History_fkey FOREIGN KEY (gct_user_id) REFERENCES gct_user(id);

-- join table for portfolio
CREATE TABLE portfolio (
  gct_user_id integer NOT NULL,
  address_id integer NOT NULL
);

-- Composite primary key
ALTER TABLE portfolio ADD CONSTRAINT portfolio_pkey PRIMARY KEY (gct_user_id, address_id);
ALTER TABLE portfolio ADD CONSTRAINT portfolio_gct_user_fkey FOREIGN KEY (gct_user_id) REFERENCES gct_user(id);
ALTER TABLE portfolio ADD CONSTRAINT portfolio_address_information_fkey FOREIGN KEY (address_id) REFERENCES address_information(id);
