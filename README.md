# openvpn_exporter
OpenVPN server statistic exporter for prometheus monitoring


Defaults:
- port 9509
- status.log path is "/var/log/status.log"

Options:
```
  -listenaddr string
        ovpnserver_exporter listen address (default ":9509")
  -metricspath string
        URL path for surfacing collected metrics (default "/metrics")
  -ovpn.log string
        Absolute path for OpenVPN server log (default "/var/log/status.log")
```
Example usage:
```
./openvpn_exporter -listenaddr 127.1:9509 -metricspath /metrics -ovpn.log /var/log/status.log
```

Docker image :
```
https://hub.docker.com/r/vaganovni/openvpn_exporter/
```

Docker usage :
```
docker run -d -p 9509:9509 -v "/var/log/status.log:/var/log.status.log" vaganovni/openvpn_exporter
```
