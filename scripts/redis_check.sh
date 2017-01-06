#!/usr/bin/env expect

spawn nc -w 2 $env(ADDR) $env(PORT)

set EC 0

send "PING\r\n"
expect "+PONG"
send "QUIT\r\n"
expect "+OK"
exit $EC