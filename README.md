# MultiOTP Web Portal for LDAP users

Just to show LDAP users' their QR code and reissue it if needed.

* It was build to use in Windows with Windows version of MultiOTP.
* It may be used in Linux(not tested).

<h2>Thanks</h2>

<h3>Web Server</h3>

**With big thanks to Alex Edwards and his wonderful book: Let's Go(https://lets-go.alexedwards.net/)!**

Web Server logic and all UI static files are taken from this book.

<h3>Overlay JS</h3>

For JS with overlay logic thanks to MS Bing Copilot)).

<h2>Requirements</h2>

* The app must be located on the same host as MultiOTP service
* The app demands MySQL db. Check "DB" section below
* The app uses TLS for web, so you need cert file and key file
* The app uses embed data file, described in 'data/data.json'
* The works with LDAP users only:
    * One domain to auth users in portal
    * Another is the domain MultiOTP bound with(<b>may be the same as for user auth</b>)
* All domains'(both auth & MultiOTP) FQDN must be resolved by local DNS(or use hosts file)
* All domains' must support LDAPS protocol

* Go version: 1.24
* Tested on MultiOTP version: 5.9.8.0. Must work on later releases(but not tested yet).

<h2>DB</h2>

Used MySQL Db. Must be installed and set.
DB user and DB pass must be set in OS env:
- OTP_DB_USR
- OTP_DB_PASS

Db must be on local server(for now). 

"otpportal" is default name for db. 
<b>Use -db flag to change default DB name</b>

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

<h2>Flags</h2>

* logDir - directory for logs; default is "logs_OTP-Portal" in the same dir as exe
* keep-logs - number of most recent logs to keep; default is 7
* addr - address of server; default is ":3000" which means localhost:3000localhost:3000"
* tls-cert - path to tls cert file; default is "tls/cert.pem" in the same dir as exe
* tls-key - path to tls key file; default is "tls/key.pem" in the same dir as exe
* db - MySQL db name; default is "otpportal"
* m := MultiOPT exe path; default is "c:/MultiOTP/windows/multiotp.exe"
* lang - language for all html pages; default is "ru"; other language available is "en"(english)
* 2fa - Use (PrivacyIdea API) provider for second factor auth

<h3>data/data.json</h3>

Example of data/data.json

```
{
    "userDomainFQDN": "<YOUR FQDN TO AUTH USER>",
	"userDomainBaseDN": "<YOUR DOMAIN BASE DN TO AUTH USER",
	"qrDomainFQDN": "<YOUR MULTI-OTP FQDN>",
	"qrDomainBaseDN": "<YOUR MULTI-OTP BASE DN>",
	"qrDomainBindUser": "<YOUR MULTI-OTP BIND USER>",
	"qrDomainBindUserPass": "<YOUR MULTI-OTP BIND USER PASS>",
    "dBUsr": "",
    "dbPass": "",
    "mfaUrl": "<YOUR PRIVACYIDEA BASE URL TO AUTH USER(OTP)>",
    "mfaTriggerUser": "<YOUR PRIVACYIDEA TRIGGER USER(ADMIN) TO AUTH USER>",
    "mfaTriggerUserPass": "<YOUR PRIVACYIDEA TRIGGER USER(ADMIN) PASS>"
}
```

data/data.json will be embed using <b>data/dataembed.go</b>.

Rename <b>data_BLANK.json</b> to data.json before use!

<h2>2fa</h2>

For second factor auth PrivacyIdea(3.10.x) API is used.

Thus you need trigger admin user added in your PrivacyIdea - and must have rights(policy) of Action: <b>token -> tokenlist=true</b>

Valid OTP is 6x number all digits string.

<h2>Localisation</h2>

Only Russian & English. Russian is default.
Control with flag "lang" ("ru" or "en").

<h2>Workflow</h2>

1) User(domain user) tries to authenticate USER_DOM_ fqdn, basedn and login/pass creds.

The app tries to do LDAPS bind(and 2fa(OTP) is -2fa flag is set), if success:

get user's DisplayName attribute(to show on page) and -> login to page with their QR.

2) To get user's QR on qr/view page:

The tries to do bind with QR_DOM_ fqdn, basedn, bindUser and bindUser pass.

If ok - the app tries to do search user's samaAccName with such filter:
```
"(&(objectClass=user)(samaccountname=*%s))"
```

Where %s is user's USER_DOM account.

If samaAccName found use it to search otpauth:// URL using MultiOTP cli:
```
> multiotp -urllink user
```

* If found - generate QR's svg and paste it into template.
* If not - print "NOT FOUND !" in QR placeholder of page.

3) To reissue using MultiOTP cli:
```
multiotp -delete user
multiotp -ldap-users-sync
```
and then again get current QR logic.