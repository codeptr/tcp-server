package tcp_server

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
)

// Client holds info about connection
type Client struct {
	conn   net.Conn
	Server *Server
}

// TCP server
type Server struct {
	address                  string // Address to open connection: localhost:9999
	config                   *tls.Config
	onNewClientCallback      func(c *Client)
	onClientConnectionClosed func(c *Client, err error)
	onNewMessage             func(c *Client, message string)
	listener                 net.Listener
	quit                     chan struct{}
}

// Read client data from channel
func (c *Client) listen() {
	c.Server.onNewClientCallback(c)
	reader := bufio.NewReader(c.conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			c.conn.Close()
			c.Server.onClientConnectionClosed(c, err)
			return
		}
		c.Server.onNewMessage(c, message)
	}
}

// Send text message to client
func (c *Client) Send(message string) error {
	return c.SendBytes([]byte(message))
}

// Send bytes to client
func (c *Client) SendBytes(b []byte) error {
	_, err := c.conn.Write(b)
	if err != nil {
		c.conn.Close()
		c.Server.onClientConnectionClosed(c, err)
	}
	return err
}

func (c *Client) Conn() net.Conn {
	return c.conn
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Called right after server starts listening new client
func (s *Server) OnNewClient(callback func(c *Client)) {
	s.onNewClientCallback = callback
}

// Called right after connection closed
func (s *Server) OnClientConnectionClosed(callback func(c *Client, err error)) {
	s.onClientConnectionClosed = callback
}

// Called when Client receives new message
func (s *Server) OnNewMessage(callback func(c *Client, message string)) {
	s.onNewMessage = callback
}

// Listen starts network server
func (s *Server) Listen() {
	var err error
	if s.config == nil {
		s.listener, err = net.Listen("tcp", s.address)
	} else {
		s.listener, err = tls.Listen("tcp", s.address, s.config)
	}
	if err != nil {
		log.Fatal("Error starting TCP server.\r\n", err)
	}
	defer s.listener.Close()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Fatal("Error accepting connection.\r\n", err)
			}
			continue
		}
		client := &Client{
			conn:   conn,
			Server: s,
		}
		go client.listen()
	}
}

// Close server
func (s *Server) Close() error {
	close(s.quit)
	return s.listener.Close() // To trigger an error in listen accept loop
}

// Creates new tcp server instance
func New(address string) *Server {
	//log.Println("Creating server with address", address)
	server := &Server{
		address: address,
		quit:    make(chan struct{}),
	}

	server.OnNewClient(func(c *Client) {})
	server.OnNewMessage(func(c *Client, message string) {})
	server.OnClientConnectionClosed(func(c *Client, err error) {})

	return server
}

func NewWithTLS(address, certFile, keyFile string) *Server {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal("Error loading certificate files. Unable to create TCP server with TLS functionality.\r\n", err)
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	server := New(address)
	server.config = config
	return server
}
