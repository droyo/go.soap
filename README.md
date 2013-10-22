A very simple SOAP 1.1 library.

Currently this library is simply a wrapper around `encoding/xml`. The only value it adds is
the flattening of multi-reference values returned by pesky services such as Apache Axis.
