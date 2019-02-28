godhcprev
=========

A simple stateless DNS server to serve forward/reverse DNS records for dynamic clients.

    deadbeef21374242.example.com 
                  |   ^
             AAAA |   | IN PTR
                  V   |
    2a00:0000:0000:0000:dead:beef:2137:4242

For now, support IPv6. Legacy (v4) support might come some day.

This is not very good code. Don't use this. It does have some tests, though.
