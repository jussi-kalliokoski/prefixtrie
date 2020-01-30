// Package prefixtrie provides a prefix trie data structure for building
// efficient indices of substring-searchable string data.
//
// The provided implementation only supports storing ints, which should be
// sufficient for most searches over slices (e.g. find elements with name
// matching a substring) and word searches for documents (the end of the word
// can be scanned from the beginning of the word).
package prefixtrie

import (
	"strings"
	"unicode/utf8"
)

// Trie is a prefix trie where you can add values and find values.
//
// A zero value Trie is ready to use.
//
// Parallel reads (Find) are safe but writes (Add) in parallel with reads or
// other writes are undefined behavior.
type Trie struct {
	root node
}

// Add adds a value to the Trie by a key.
func (t *Trie) Add(key string, value int) {
	for i := range key {
		t.root.add(key[i:], value)
	}
}

// Find finds the values that were added with a key that is a substring match
// of the prefix.
//
// The found values are appended to the dst slice and the resulting slice is
// returned. Passing nil as dst will return the values in a newly allocated
// slice. If no matching values are found, dst will be returned as is.
func (t *Trie) Find(dst []int, prefix string) []int {
	return t.root.find(dst, prefix)
}

type node struct {
	prefix   string
	values   []int
	children []node
}

func (n *node) add(key string, value int) {
	commonPrefix := n.commonPrefix(n.prefix, key)
	if len(commonPrefix) < len(n.prefix) {
		n.split(commonPrefix)
	}
	if len(commonPrefix) == len(key) {
		n.values = append(n.values, value)
		return
	}
	// sorted add (ascending rune value)
	subKey := key[len(commonPrefix):]
	firstRune := n.firstRune(subKey)
	for i := range n.children {
		c := &n.children[i]
		if firstRuneOfChild := n.firstRune(c.prefix); firstRuneOfChild == firstRune {
			c.add(subKey, value)
			return
		} else if firstRuneOfChild > firstRune {
			n.insertChildAtIndex(node{prefix: subKey, values: []int{value}}, i)
			return
		}
	}
	n.children = append(n.children, node{prefix: subKey, values: []int{value}})
}

func (n node) find(dst []int, prefix string) []int {
	if strings.HasPrefix(n.prefix, prefix) {
		// match found
		return n.collectValues(dst)
	}
	// binary search for the child with matching first rune of prefix
	// adapted and inlined from sort.Search
	subPrefix := prefix[len(n.prefix):]
	firstRune := n.firstRune(subPrefix)
	lo, hi := 0, len(n.children)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if n.firstRune(n.children[mid].prefix) < firstRune {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo >= len(n.children) {
		return dst
	}
	if n.firstRune(n.children[lo].prefix) != firstRune {
		return dst
	}
	return n.children[lo].find(dst, subPrefix)
}

func (n node) collectValues(dst []int) []int {
	dst = append(dst, n.values...)
	for _, c := range n.children {
		dst = c.collectValues(dst)
	}
	return dst
}

func (n *node) split(commonPrefix string) {
	*n = node{
		prefix: commonPrefix,
		children: []node{
			{
				prefix:   n.prefix[len(commonPrefix):],
				values:   n.values,
				children: n.children,
			},
		},
	}
}

func (n *node) insertChildAtIndex(c node, i int) {
	n.children = append(n.children, n.children[len(n.children)-1])
	copy(n.children[i+1:], n.children[i:len(n.children)-1])
	n.children[i] = c
}

func (*node) firstRune(str string) rune {
	r, _ := utf8.DecodeRuneInString(str)
	return r
}

// commonPrefix returns the the prefix that the two strings have in common.
//
// valid UTF-8 strings are assumed.
func (*node) commonPrefix(a, b string) string {
	i, commonLen := 0, len(a)
	if len(b) < commonLen {
		commonLen = len(b)
	}
	for i < commonLen {
		ra, _ := utf8.DecodeRuneInString(a[i:])
		rb, size := utf8.DecodeRuneInString(b[i:])
		if ra != rb {
			break
		}
		i += size
	}
	return a[:i]
}
