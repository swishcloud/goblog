mkdir .cache
#generate TLS certificate
"C:\Program Files (x86)\OpenSSL-1.1.1h_win32\openssl.exe" req -newkey rsa:4096 ^
-x509 ^
-sha256 ^
-days 365 ^
-nodes ^
-out .cache/localhost.crt ^
-keyout .cache/localhost.key ^
-subj "/C=CH/ST=GUANGDNG/L=SHENZHEN/O=SECURITY/OU=IT DEPARTMENT/CN=localhost"
REM set up and run database
REM docker-compose -p goblog-project -f docker-compose-postgres.yaml up -d