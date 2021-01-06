package mpt

import (
	"fmt"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
)

type (
	branch struct {
		nodeBase
		nibbles [16]node
		value   trie.Object
	}
)

// changeState change state from passed state which is returned state by nibbles
func (br *branch) changeState(s nodeState) {
	if s == dirtyNode && br.state != dirtyNode {
		br.state = dirtyNode
	}
}

func (br *branch) flush() {
	if br.value != nil {
		br.value.Flush()
	}
}

func (br *branch) serialize() []byte {
	if br.state == dirtyNode {
		br.serializedValue = nil
		br.hashedValue = nil
	} else if br.serializedValue != nil { // not dirty & has serialized value
		return br.serializedValue
	}

	var serializedNodes []byte
	var serialized []byte
	for i := 0; i < 16; i++ {
		switch br.nibbles[i].(type) {
		case nil:
			serialized = encodeByte(nil)
		default:
			if serialized = br.nibbles[i].serialize(); hashableSize <= len(serialized) {
				serialized = encodeByte(br.nibbles[i].hash())
			}
		}
		serializedNodes = append(serializedNodes, serialized...)
	}

	if br.value == nil {
		serialized = encodeList(serializedNodes, encodeByte(nil))
	} else {
		// value of branch does not use hash
		serialized = encodeList(serializedNodes, encodeByte(br.value.Bytes()))
	}
	br.serializedValue = make([]byte, len(serialized))
	copy(br.serializedValue, serialized)
	br.hashedValue = nil
	br.state = serializedNode

	if printSerializedValue {
		fmt.Println("serialize branch : ", serialized)
	}
	return serialized
}

func (br *branch) hash() []byte {
	if br.state == dirtyNode {
		br.serializedValue = nil
		br.hashedValue = nil
	} else if br.hashedValue != nil { // not dirty & has hashed value
		return br.hashedValue
	}

	serialized := br.serialize()
	serializedCopy := make([]byte, len(serialized))
	copy(serializedCopy, serialized)
	digest := calcHash(serializedCopy)

	br.hashedValue = make([]byte, len(digest))
	copy(br.hashedValue, digest)
	br.state = serializedNode

	if printHash {
		fmt.Printf("hash branch : <%x>\n", digest)
	}

	return digest
}

func (br *branch) addChild(m *mpt, k []byte, v trie.Object) (node, nodeState) {
	if len(k) == 0 {
		if v.Equal(br.value) == true {
			return br, br.state
		}
		br.value = v
		br.state = dirtyNode
		return br, dirtyNode
	}
	dirty := dirtyNode
	if br.nibbles[k[0]] == nil {
		br.nibbles[k[0]], dirty = m.set(nil, k[1:], v)
	} else {
		br.nibbles[k[0]], dirty = br.nibbles[k[0]].addChild(m, k[1:], v)
	}
	br.changeState(dirty)
	return br, br.state
}

func (br *branch) deleteChild(m *mpt, k []byte) (node, nodeState, error) {
	//fmt.Println("branch deleteChild : k ", k)
	var nextNode node
	if len(k) == 0 {
		br.value = nil
		br.state = dirtyNode
	} else {
		dirty := dirtyNode
		if br.nibbles[k[0]] == nil {
			return br, br.state, nil
		}

		if nextNode, dirty, _ = br.nibbles[k[0]].deleteChild(m, k[1:]); dirty != dirtyNode {
			return br, br.state, nil
		}
		br.nibbles[k[0]] = nextNode
		br.changeState(dirty)
	}

	// check remaining nibbles on n(current node)
	// 1. if n has only 1 remaining node after deleting, n will be removed and the remaining node will be changed to extension.
	// 2. if n has only value with no remaining node after deleting, node must be changed to leaf
	// Branch has least 2 nibbles before deleting so branch cannot be empty after deleting
	remainingNibble := 16
	for i, nn := range br.nibbles {
		if nn != nil {
			if remainingNibble != 16 { // already met another nibble
				remainingNibble = -1
				break
			}
			remainingNibble = i
		}
	}

	//If remainingNibble is -1, branch has 2 more nibbles.
	if remainingNibble != -1 {
		if br.value == nil {
			// check nextNode.
			// if nextNode is extension or branch, n must be extension
			n := br.nibbles[remainingNibble]
			if v, ok := br.nibbles[remainingNibble].(hash); ok {
				serializedValue, _ := m.bk.Get(v)
				n = deserialize(serializedValue, m.objType, m.db)
			}
			switch nn := n.(type) {
			case *extension:
				return &extension{sharedNibbles: append([]byte{byte(remainingNibble)}, nn.sharedNibbles...),
					next: nn.next, nodeBase: nodeBase{state: dirtyNode}}, dirtyNode, nil
			case *branch:
				return &extension{sharedNibbles: []byte{byte(remainingNibble)}, next: nn,
					nodeBase: nodeBase{state: dirtyNode}}, dirtyNode, nil
			case *leaf:
				return &leaf{keyEnd: append([]byte{byte(remainingNibble)}, nn.keyEnd...), value: nn.value,
					nodeBase: nodeBase{state: dirtyNode}}, dirtyNode, nil
			default:
				log.Panicf("Not considered nn = %v\n", nn)
			}
		} else if remainingNibble == 16 {
			return &leaf{value: br.value, nodeBase: nodeBase{state: dirtyNode}}, dirtyNode, nil
		}
	}
	return br, dirtyNode, nil
}

func (br *branch) get(m *mpt, k []byte) (node, trie.Object, error) {
	var result trie.Object
	var err error
	if len(k) != 0 {
		if br.nibbles[k[0]] == nil {
			return br, nil, nil
		}
		br.nibbles[k[0]], result, err = br.nibbles[k[0]].get(m, k[1:])
	} else {
		result = br.value
	}

	return br, result, err
}

func (br *branch) proof(m *mpt, k []byte, depth int) ([][]byte, bool) {
	var proofBuf [][]byte
	result := true
	if len(k) != 0 {
		proofBuf, result = br.nibbles[k[0]].proof(m, k[1:], depth+1)
		if result == false {
			return nil, false
		}
		buf := br.serialize()
		if len(buf) < hashableSize && depth != 1 {
			return nil, true
		}

		if proofBuf == nil {
			proofBuf = make([][]byte, depth+1)
		}
		proofBuf[depth] = buf
	} else {
		// find k
		buf := br.serialize()
		if len(buf) < hashableSize && depth != 1 {
			return nil, true
		}
		proofBuf = make([][]byte, depth+1)
		proofBuf[depth] = buf
		return proofBuf, result
	}
	return proofBuf, result
}
