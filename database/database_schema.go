package database

var schema = map[string]string{
	"gct_user": `CREATE TABLE gct_user (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  		name text NOT NULL,
  		password text NOT NULL,
 		inserted_at DATETIME NOT NULL,
 		amended_at DATETIME NOT NULL
	  );`,

	"gct_config": `CREATE TABLE gct_config (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		config_name text NOT NULL,
		config_full BLOB NOT NULL,
		inserted_at DATETIME NOT NULL,
		amended_at DATETIME NOT NULL,
		gct_user_id integer NOT NULL,
  		FOREIGN KEY(gct_user_id) REFERENCES gct_user(id) 
	);`,

	"order_history": `CREATE TABLE order_history (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		order_id text NOT NULL,
		fulfilled_on DATETIME NOT NULL,
		currency_pair text NOT NULL,
		asset_type text NOT NULL,
		order_type text NOT NULL,
		amount real NOT NULL,
		rate real NOT NULL,
		exchange_name text NOT NULL,
		inserted_at DATETIME NOT NULL,
		amended_at DATETIME NOT NULL
	);`,

	"exchange_trade_history": `CREATE TABLE exchange_trade_history (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		fulfilled_on DATETIME NOT NULL,
		currency_pair text NOT NULL,
		asset_type text NOT NULL,
		order_type text NOT NULL DEFAULT "NOT SPECIFIED",
		amount real NOT NULL,
		rate real NOT NULL,
		order_id integer NOT NULL,
		exchange_name text NOT NULL,
		inserted_at DATETIME NOT NULL,
		amended_at DATETIME NOT NULL
	);`}

var deprecatedDatabaseTables = []string{}
