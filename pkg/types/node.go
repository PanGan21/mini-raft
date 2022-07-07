package types

type Node struct {
	connect bool
	address string
}

func NewNode(address string) *Node {
	return &Node{
		address: address,
	}
}
