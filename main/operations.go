package main

import "math/big"

// input the identifier of the node, 0-63
// return the start of the interval
func (node *Node) FingerStart(nodeId int) *big.Int {
	id := node.Identifier
	id = new(big.Int).Add(id, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(nodeId)-1), nil))
	return new(big.Int).Mod(id, hashMod)
}
