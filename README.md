Or just do a full WebRTC solution like home assistant?

The first idea was to have HA proxy and then proxy to port served on a wireguard network.

Examples:

```shell
# Expose a web server running on localhost to the remote host
tunnel --private-key priv --expose 127.0.0.1:8080 --peer-public-key pub server:5123
tunnel -l 5123 --private-key priv --expose 127.0.0.1:8080 --peer-public-key pub
```

```shell
tunnel --expose 192.168.1.100:8080 --eexpose 192.168.1.100:443 server:5123
# Listen on server
tunnel -l 5123 -e 8080:localhost:8080
```

TODO: Issue is that we're trying to do TUN stuff. If we know the peer, just do
noisy sockets instead?
