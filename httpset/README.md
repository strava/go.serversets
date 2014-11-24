go.serversets/httpset [![Godoc Reference](https://godoc.org/github.com/strava/go.serversets?status.png)](https://godoc.org/github.com/strava/go.serversets/httpset)
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

		cluster := httpset.New(watch)
		cluster.UseHTTPS = true  // if scheme not specified, will use https

		// can make requests using the standard net/http methods
		resp, err = cluster.Do(req)

		// use just the path and the hostname will be added
		// scheme will be http or https depending on cluster.UseHTTPS
		resp, err = cluster.Get("/some/path")

		// or use a complete url and only the hostname will be swapped
		resp, err = cluster.Head(http://somehost.com/some/path)

		resp, err = cluster.Post(url, bodyType, body)
		resp, err = cluster.PostForm(url)

	    // the hostnames of these requests are replaces with those in the serverset
	    // in a round-robin fashion.
	}

Dependencies
------------
* [github.com/strava/go.serversets](github.com/strava/go.serversets) to get the server list

Potential Improvements and Contributing
---------------------------------------
More better than round-robin. If you'd like, submit a pull request.
