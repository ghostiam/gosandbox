# NFQUEUE

**WARNING: NFQUEUE only works on Linux**

## Install dependencies
```bash
sudo apt install libnetfilter-queue-dev libsctp-dev
```

## Setup

for local:
```bash
sudo iptables -A INPUT -j NFQUEUE --queue-num 0
sudo iptables -A OUTPUT -j NFQUEUE --queue-num 0
```

only for forwarding:
```bash
sudo sysctl -w net.ipv4.ip_forward=1
sudo iptables -t nat -A POSTROUTING -o ens33 -j MASQUERADE
sudo iptables -t mangle -A POSTROUTING -j NFQUEUE --queue-num 0
```

## Running
```bash
sudo go run nfqueue.go
```

and try:
```bash
curl http://httpbin.org/get?test=nfqueue
```
returned:
```json
{
  "args": {
    "test": "replace"
  }, 
  "headers": {
    "Accept": "*/*", 
    "Host": "httpbin.org", 
    "User-Agent": "curl/7.58.0"
  }, 
  "origin": "<IP>", 
  "url": "http://httpbin.org/get?test=replace"
}
```
