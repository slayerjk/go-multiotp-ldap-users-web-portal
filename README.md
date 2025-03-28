# go-multiotp-ldap-users-portal
MultiOTP Web Portal for LDAP users

<h2>Localisation</h2>

Only Russian & English. Russian is default.
Control with flag "lang" ("ru" or "en").

<h2>DB</h2>

Used MySQL Db. Must be installed and set.
DB user and DB pass must be set in OS env:
- OTP_DB_USR
- OTP_DB_PASS

Db must be on local server(for now). 

"otpportal" is default name for db.

Sqlscript:
```
CREATE DATABASE <DBNAME> CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE <DBNAME>;

CREATE TABLE sessions (
    token CHAR(43) PRIMARY KEY,
    data BLOB NOT NULL,
    expiry TIMESTAMP(6) NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

CREATE USER '<OTP_DB_USR>'@'localhost';

GRANT SELECT, INSERT, UPDATE, DELETE ON <DBNAME>.* TO '<OTP_DB_USR>'@'localhost';

ALTER USER '<OTP_DB_USR>'@'localhost' IDENTIFIED BY '<OTP_DB_PASS>';
```

<h2>Workflow</h2>

**With big thanks to wonderful book of Alex Edwards: Let's Go(https://lets-go.alexedwards.net/)!**