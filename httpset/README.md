go.serversets/httpset [![Build Status](https://travis-ci.org/strava/go.serversets.png?branch=master)](https://travis-ci.org/strava/go.serversets) [![Godoc Reference](https://godoc.org/github.com/strava/go.serversets?status.png)](https://godoc.org/github.com/strava/go.serversets/httpset)
=====================

Package **httpset** provides round-robin balancing over a set of endpoints 
provided by [go.serversets](/..). Connection reuse is handled by the 'net/http'
standard library.

Usage
-----

	package main

	import (
		"log"

		"github.com/strava/go.serversets"
		"github.com/strava/go.serversets/httpset"
	)

	func main() {
		zookeepers := []string{"10.0.1.0", "10.0.5.0", "10.0.9.0"}
		watch, err := serversets.New(serversets.Production, "service_name", zookeepers).Watch()
		if err != nil {
			// This will be a problem connecting to Zookeeper
			log.Fatalf("Registration error: %v", err)
		}

		t := httpset.NewTransport(watch)
		t.UseHTTPS = true  // if scheme not specified, will use https

		client := &http.Client{
			Transport: t,
		}

		// Use the client as you normally would.
	}

Dependencies
------------
* [github.com/strava/go.serversets](github.com/strava/go.serversets) to get the server list.
However, one can use a predefined set of servers by doing something like:

		t := httpset.NewTransport(nil)
		t.SetEndpoints([]string{"server1.com", "server2.com"})

Potential Improvements and Contributing
---------------------------------------
More better than round-robin. If you'd like, submit a pull request.
