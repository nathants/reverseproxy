# reverseproxy

## why

simple reverse proxy should be easy

## what

a reverse proxy allowing multiple upstreams by host name

## install

```
go get github.com/nathants/reverseproxy
```

## usage

```
>>reverseproxy -h

Usage: main [--addr ADDR] [--timeout TIMEOUT] [--ssl-cert SSL-CERT] [--ssl-key SSL-KEY] [--upstream UPSTREAM]

Options:
  --addr ADDR,         -a ADDR [default: :443]
  --timeout TIMEOUT,   -t TIMEOUT [default: 5]
  --ssl-cert SSL-CERT, -c SSL-CERT
  --ssl-key SSL-KEY,   -k SSL-KEY
  --upstream UPSTREAM, -u UPSTREAM
                         may specify multiple times. --upstream example.com=localhost:8080
```
