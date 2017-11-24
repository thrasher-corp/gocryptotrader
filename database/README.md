## GoCryptoTrader database
A backend GoCryptoTrader application. It is developed with postgreSQL 9.5.10 and is using SQLBoiler v2.5.1

Big thank you to the team at volatiletech for providing this tool https://github.com/volatiletech/sqlboiler

## This is still in active development
You can track ideas, planned features and what's in progresss on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

## Current Features
None (Hopefully soon!)

## How to use
In postgreSQL

  CREATE user "gocryptotrader" with password as "lol123"
  CREATE database "gocryptotrader"
  INSERT tables using the trading.sql file

Models folder can be completely deleted and fully regenerated with go generate
and SQLBoiler
