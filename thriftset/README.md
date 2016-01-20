go.serversets/thriftset [![Build Status](https://travis-ci.org/strava/go.serversets.png?branch=master)](https://travis-ci.org/strava/go.serversets) [![Godoc Reference](https://godoc.org/github.com/strava/go.serversets?status.png)](https://godoc.org/github.com/strava/go.serversets/thriftset)
=====================

Package **thriftset** provides "least active request" balancing over a set of endpoints
provided by [go.serversets](/..). Connections are kept in a pool and reused as needed.

Usage
-----

	zookeepers := []string{"10.0.1.0", "10.0.5.0", "10.0.9.0"}
	watch, err := serversets.New(serversets.Production, "service_name", zookeepers).Watch()
	if err != nil {
		// This will be a problem connecting to Zookeeper
		log.Fatalf("Registration error: %v", err)
	}

	ts := thriftset.New(watch)

	// or using a fixed set of servers, to just use the loadbalancing
	ts := thriftset.New(fixedset.New([]string{"host1:1234", "host2:1234"}))

	service := New(ts)

	resp, err := service.Find(123)
	log.Printf("%v %v", resp, err)

The Service object is a helpful way to wrap the thrift interface into something nicer.
It handles the "checking out" or connections and returning them. This `thriftset` package
returns a connection to the endpoint with the least amount of of checked out connections.

	// A Service object wraps the thrift interface with helper methods to fetch/update data.
	type Service struct {
		set              *thriftset.ThriftSet
		protocolFactory  thrift.TProtocolFactory
		transportFactory thrift.TTransportFactory
	}

	func New(set *thriftset.ThriftSet) (*Service) {
		return &Service{
			set:              set,
			protocolFactory:  thrift.NewTBinaryProtocolFactory(),
			transportFactory: thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory()),
		}
	}

	func (s *Service) Close() error {
		return s.set.Close()
	}

	func (s *Service) getClient() (*thriftset.Conn, *sampleservice.SampleServiceClient, error) {
		conn, err := s.set.GetConn()
		if err != nil {
			return nil, nil, err
		}

		if conn.Client == nil {
			transport := s.transportFactory.GetTransport(conn.Socket)
			err = transport.Open()
			if err != nil {
				return nil, nil, err
			}

			// The client factory is cached on the client
			conn.Client = sampleservice.NewSampleServiceClientFactory(transport, s.protocolFactory)
		}

		return conn, conn.Client.(*sampleservice.SampleServiceClient), nil
	}

	func (s *Service) releaseClient(conn *thriftset.Conn, err error) error {
		// do some sort of check to see if this error should close the connection or not.
		// ie. is it network related, or just a thrift exception
		if fatalError(err) {
			conn.Close()
			return err
		}

		return conn.Release() // returns the connection back to the pool
	}

	func (s *Service) Find(id int64) (*Sample, error) {
		req := sampleservice.NewGetRequest()
		req.ID = id

		conn, client, err := s.getClient()
		if err != nil {
			return nil, err
		}

		resp, err := client.Get(req)
		err = s.releaseClient(conn, err)
		if err != nil {
			return nil, err
		}

		// convert the thrift object into a "clearer" go type.
		// Maybe add some validation.
		return &Sample{
			ID: id
			Metadata: resp.Metadata,
		}, nil
	}

Potential Improvements and Contributing
---------------------------------------
It'd be nice to mark an endpoint as down if it returns too many errors.
If you'd like, submit a pull request.
