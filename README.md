go.serversets [![Godoc Reference](https://godoc.org/github.com/strava/go.serversets?status.png)](https://godoc.org/github.com/strava/go.serversets)
=============

Package **go.serversets** provides an simple interface for service discovery using [Apache Zookeeper](http://zookeeper.apache.org/).
Servers/endpoints register themselves and clients always have an updated host list. 

This core package just provides a list of hostnames and ports. Sub-packages wrap 
the endpoint list for common use cases:

* [mcset](/mcset) provides consistent hashing over a set of memcache hosts.
* [httpset](/httpset) round-robins standard HTTP requests to the set of hosts.

This package is used internally at [Strava](http://strava.com) for 
[Finagle](https://twitter.github.io/finagle/) service discovery and memcache node registration. 

Usage
-----
First, create a ServerSet defining an environment and service name.

	zookeepers := []string{"zk01.internal", "zk02.internal", "zk03.internal"}
	serverSet := serversets.New(serversets.Staging, "service_name", zookeepers)

This doesn't actually connect to the Zookeeper servers, just defines the namespace.
Methods like `RegisterEndpoint` and `Watch` will return an error if they can't connect.
Available environment constants: `serversets.Local`, `serversets.Staging`, `serversets.Production` and `serversets.Test`.

### Register an Endpoint

Service endpoints/producers/servers should register themselves as an endpoint. Example:
	
	pingFunction := func() error {
		return nil
	}

	endpoint, err := serverSet.RegisterEndpoint(
		localIP,
		servicePort,
		pingFunction)

The ping function can be `nil`. But if it's not, it'll be checked every second, by default. If there is an 
error the endpoint will be unregistered. Once the issue is resolved it'll be reregistered automatically.
This allows for registering external processes that may fail independently of the monitoring process.

### Watch the list of available endpoints, for consumers

	watch, err = serverSet.Watch()
	if err != nil {
		// probably something wrong with connecting to Zookeeper
		panic(err)
	}

	endpoints := watch.Endpoints()
	for {
		<-watch.Event()
		// endpoint list changed
	}

The `watch.Event()` channel will be triggered whenever the endpoint list changes
and `watch.Endpoints()` will contain the updated list of available endpoints.

Finagle Compatibility
---------------------
The Zookeeper zNode data is designed to be compatible with [Finagle](https://twitter.github.io/finagle/) ServerSets.
It is just a matter or matching the namespaces in the ServerSet declarations.
This library registers endpoints to zNodes similar to `/discovery/staging/service_name/member_0000000318`.

A Scala snippet to register an endpoint discoverable by the watch created above:

	val serverHost: java.net.InetSocketAddress
	val zookeeperHost: java.net.InetSocketAddress

	val zookeeperClient = new ZookeeperClient(sessionTimeout, zookeeperHost)
	val serverSet = new ServerSetImpl(zk.zookeeperClient, "/discovery/staging/service_name")
	val cluster = new ZookeeperServerSetCluster(serverSet)

	cluster.join(serverHost)

The namespaces used by this library are completely configurable. One just needs to defining their own `BaseZnodePath` function.

Dependencies
------------
The centralized and consistent storage of service endpoints is handled by [Apache Zookeeper](http://zookeeper.apache.org/).
So obviously, such a cluster is required. Go package dependencies beyond the standard library include:

* [github.com/samuel/go-zookeeper/zk](https://github.com/samuel/go-zookeeper/)

Tests
-----
Tests require a Zookeeper server. The default is "localhost" but a different 
host can be used by changing the `TestServer` variable in [serverset_test.go](serverset_test.go)

	go test github.com/strava/go.serversets/...

Potential Improvements and Contributing
---------------------------------------
This library simply provides a list of active endpoints. But it would nice if it did some
load balancing, error checking, retries etc. Simple versions of this are available for 
[memcache](/mcset) and [http](/httpset) but they can be improved. 
So, if you have some ideas, submit a pull request.
