httptunnel
==========

Super simple standalone http proxy for i2p using SAM. It's mostly just a way of
easily setting up a single-identity for a single application. It offers a single
HTTP Proxy tunnel for a single application and is a form of isolation, but it
isn't Tor-like isolation in that it's intended to be started by the application
and is not based on a SOCKS proxy with extended behavior. It's tiny, it's pure
Go, it's easy to embed in other HTTP applications requiring a client proxy, and
the default executable is self-supervising on Unix-like plaforms. It will work
with any i2p router that has a SAM bridge.

