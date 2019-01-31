package kobject

// External holds external service definition
type External struct {
	host string  `compose:"host"`
	Port []Ports `compose:"ports"`
}

// HTTPGetAction describes an action based on HTTP Get requests.
type HTTPGetAction struct {
	// Optional: Path to access on the HTTP server.
	Path string `json:"path,omitempty"`
	// Required: Name or number of the port to access on the container.
	Port int `json:"port,omitempty"`
}

// TCPSocketAction describes an action based on opening a socket
type TCPSocketAction struct {
	// Required: Port to connect to.
	Port int `json:"port,omitempty"`
}
