# sentinel

Go-sak som sjekker status til applikasjonene våre og poster dette til InfluxDB

-Henter URLene fra Fasit-    (kan begynne med å statisk konfigurere de)

https://vera.adeo.no/isAlive
https://sera.adeo.no/isAlive
https://fasit.adeo.no/isAlive
https://basta.adeo.no/isAlive
https://influxdb.adeo.no/healthcheck...

sjekker i utgangspunktet kun http returkode på URLen definert, hvis 200 -> status OK

3 statuser, OK, ERROR, UNKNOWN
0, 1, -1

skriver datapunktet isalive application=<app>,environment=<env> status=<status>

...til influxdb


