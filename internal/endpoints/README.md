internal/endpoints
==================

Package **endpoints** is an internal package that tries to abstract
the concept of a set of endpoints each with their own connection pool.

It currently only does "least active requests" load balancing. It would be nice to extend this
and add support for marking endpoints as down temporarily if there are issues.

Right now only the `thriftset` package uses this code, but it would be interesting to
implement a "least active requests" http load balancer using this code.
