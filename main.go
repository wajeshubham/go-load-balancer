package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	isAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	address string
	proxy   *httputil.ReverseProxy
}

func (s *simpleServer) Address() string {
	// get server address
	return s.address
}

func (s *simpleServer) isAlive() bool {
	// check if server is alive
	return true
}

func (s *simpleServer) Serve(rw http.ResponseWriter, r *http.Request) {
	// serve the request
	s.proxy.ServeHTTP(rw, r)
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.isAlive() {
		// if our server is not alive change the server with incrementing round robin count
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	// if server is alive increment the rr count and return the server
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) {
	// get the next available server
	server := lb.getNextAvailableServer()
	fmt.Printf("Forwarding request to %s\n", server.Address())
	// serve the request
	server.Serve(rw, r)
}

func newSimpleServer(address string) *simpleServer {
	serverUrl, err := url.Parse(address)
	handleErr(err)
	// create and return a simple server
	return &simpleServer{
		address: address,
		proxy:   httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	// create and return a load balancer
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("error %v\n", err)
		os.Exit(1)
	}
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.bing.com"),
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.google.com"),
	}

	lb := NewLoadBalancer("8000", servers)
	fmt.Printf("Server listening on port %s\n", lb.port)

	handleRedirect := func(rw http.ResponseWriter, r *http.Request) {
		lb.serveProxy(rw, r)
	}
	http.HandleFunc("/", handleRedirect)

	http.ListenAndServe(":"+lb.port, nil)
}
