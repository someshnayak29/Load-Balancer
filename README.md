# Load Balancer

<p align="center">
  <img src="https://i.postimg.cc/prDRd08h/logo.gif" style="border-radius:9px;"/>
</p>

**Load Balancing Algorithm**
- least-time
- weighted-round-robin
- connection-per-time
- round-robin

## Description :books:

- least-time: The load balancer will analyse and monitor the lower average latency of the provided servers and redirect the request to that server.
- weighted-round-robin: On booting the load balance, it will read from the configuration file the weight of every server and redirect the request to the server with the highest weight rate.
- connection-per-second: On booting the load balancer, it will check the number of connections on every server, and it will make that server inactive if the number of requests redirected per 1 minute exceeded the provided configuration of that server.

**The load balancer checks the server's health status every 1 minute.**
