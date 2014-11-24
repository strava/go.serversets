go.serversets/mcset [![Godoc Reference](https://godoc.org/github.com/strava/go.serversets?status.png)](https://godoc.org/github.com/strava/go.serversets/mcset)
=====================

Package **mcset** provides consistent sharding over a set of memcache nodes
provided by [go.serversets](/..).

Consistent hashing is provided by [github.com/stathat/consistent](github.com/stathat/consistent)
and the `MCSet` object fulfills the [ServerSelector interface](https://github.com/bradfitz/gomemcache/blob/master/memcache/selector.go#L30)
to be used with [github.com/bradfitz/gomemcache/memcache](github.com/bradfitz/gomemcache/memcache).


Client Usage
------------

	package main

	import (
		"log"

		"github.com/bradfitz/gomemcache/memcache"
		"github.com/strava/go.serversets"
		"github.com/strava/go.serversets/mcset"
	)

	func main() {
		zookeepers := []string{"10.0.1.0", "10.0.5.0", "10.0.9.0"}
		watch, err := serversets.New(serversets.Production, "memcache_nodes", zookeepers).Watch()
		if err != nil {
			// This will be a problem connecting to Zookeeper
			log.Fatalf("Registration error: %v", err)
		}

		cluster := mcset.New(watch)
		memcacheClient := memcache.NewFromSelector(cluster)

		// key will be consistently hash over the current set of servers
		item, err := memcacheClient.Get("key")

		log.Print(item, err)
	}

Memcache node registration
--------------------------

	package main

	import (
		"log"
		"net"
		"os"

		"github.com/bradfitz/gomemcache/memcache"
		"github.com/strava/go.serversets"
	)

	func main() {
		localHostname, err := os.Hostname()
		if err != nil {
			log.Fatalf("Unable to determine hostname: %v", err)
		}

		// define a "ping" function
		localClient := memcache.New(net.JoinHostPort(localHostname, "11211"))
		pingFunc := func() error {
			_, err := localClient.Get("ping")
			if err == nil || err == memcache.ErrCacheMiss {
				return nil
			}

			log.Printf("Memcache error: %v", err)
			return err
		}

		// register the endpoint to the zookeepers
		zookeepers := []string{"10.0.1.0", "10.0.5.0", "10.0.9.0"}
		endpoint, err := serversets.New(serversets.Production, "memcache_nodes", zookeepers).
			RegisterEndpoint(localHostname, 11211, pingFunc)

		if err != nil {
			// This will be a problem connecting to Zookeeper
			log.Fatalf("Registration error: %v", err)
		}

		log.Printf("Registered as %s:11211", localHostname)

		// will will wait forever, since you'll never actually close the endpoint
		<-endpoint.CloseEvent
	}

Dependencies
------------
* [github.com/strava/go.serversets](github.com/strava/go.serversets) to get the server list
* [github.com/stathat/consistent](github.com/stathat/consistent) for consistent hashing
