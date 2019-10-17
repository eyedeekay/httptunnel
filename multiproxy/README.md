BrowserProxy
============

*This is **emphatically unofficial** and I did it in a night because http*
*proxies are easy. It could do something you like. It could burn your house*
*down. I strongly advise you to help me with it if you want something it does*
*to become mainstream. Right now I'm a little worried about sanitization.*

This is an HTTP Proxy for I2P which introduces new behavior in a configurable
way, in particular, it is capable of controlling the *creation* and *lifespan*
of a *multiplex* of *small(1)* tunnel pools, each corresponding to an I2P
destination. What I've long ago elected to call destination isolation because
in the context of I2P, the analogous terminology from Tor introduces confusion.
The othert types of isolation from the Tor context largely already exist in I2P,
and the remaining item of concern is the "Destination," primarily via it's
relationship to the web browser.

*It does **not** create a new destination for every single site by default.*
Instead it accepts Proxy-Authorization parameters, which are used to either
spawn or select an HTTP client to use to forward messages to and from I2P. Each
client has it's own destination. This could be used in a browser extension in
conjunction with a Contextual-Identities type feature to isolate activities from
eachother in a granular way. If passed no auth parameters, it will fall back
to a "general" tunnel which behaves almost exactly as httptunnel in the parent
directory or the http proxies in I2P and i2pd.

It also has an aggressive mode, which creates a whole new tunnel pool for every
single eepSite you visit, by domain(which means that if you visit both the
base32 and readable domain, it will create *two* destinations). I advise against
using it.

Features: Done
--------------

  * Self-supervising, Self-restarting on Unixes
  * CONNECT support
  * "New Ident" signaling interface

Features: Planned
-----------------

  * Outproxy Support
  * Traffic Shaping
