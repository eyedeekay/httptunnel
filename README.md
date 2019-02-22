i2phttpproxy
============

This is a very simple standalone HTTP Proxy for i2p based on the SAM Bridge. It
has a few advantages in certain situations, especially for adapting applications
that speak HTTP to the i2p network. It allows applications to start their own
HTTP proxies, with their own identities within i2p and their own discrete
configurations. It also has some disadvantages, it cannot add new readable
names to your i2p address book nor is it able to use an outproxy. It's new, but
it should be stable enough to experiment with a Tor Browser or a hardened
Firefox configuration.

It is not, and is not intended to be, and will not be intended for use by
multiple clients at the same time. It might be more-or-less OK as part of an
inproxy but you should only use it for one client at a time. A multi-client
solution will also be available soon([eeProxy](https://github.com/eyedeekay/eeProxy).

Features: Done
--------------

  * Self-supervising, Self-restarting on Unixes
  * CONNECT support
  * "New Ident" signaling interface(Unix-only for now)(I guess I might have done
  for Windows too now but I haven't tried it out yet).
