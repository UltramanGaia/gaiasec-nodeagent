package runtime

// Client abstracts access to the container runtime API for listing containers.

type Client interface {
	ListContainers() ([]Container, error)
	RuntimeType() string
}
