package prefixtrie_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"
	"unsafe"

	"github.com/jussi-kalliokoski/prefixtrie"
)

func Test(t *testing.T) {
	var trie prefixtrie.Trie
	rand.Seed(1)
	keys := make([]string, 0, 1000)
	for cap(keys) > len(keys) {
		keys = append(keys, randomHex())
	}
	for i, key := range keys {
		trie.Add(key, i)
	}

	t.Run("full match", func(t *testing.T) {
		for i, key := range keys {
			results := trie.Find(nil, key)
			if 1 != len(results) {
				t.Log(key)
				t.Fatalf("expected len() 1 at index %d, got %d %#v", i, len(results), results)
			}
			if i != results[0] {
				t.Fatalf("expected %d, got %d", i, results[0])
			}
		}
	})

	t.Run("substring match", func(t *testing.T) {
		t.Run("from beginning", func(t *testing.T) {
			for i, key := range keys {
				reverseKey := reverse(key)
			iteratingSubstrings:
				for j := range reverseKey {
					results := trie.Find(nil, reverse(reverseKey[j:]))
					for _, result := range results {
						if result == i {
							continue iteratingSubstrings
						}
					}
					t.Fatalf("expected to find value %d starting from substring at [%d:]", i, j)
				}
			}
		})

		t.Run("from end", func(t *testing.T) {
			for i, key := range keys {
			iteratingSubstrings:
				for j := range key {
					results := trie.Find(nil, key[j:])
					for _, result := range results {
						if result == i {
							continue iteratingSubstrings
						}
					}
					t.Fatalf("expected to find value %d starting from substring at [%d:]", i, j)
				}
			}
		})
	})

	t.Run("not found", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			results := trie.Find(nil, randomHex())
			if len(results) > 0 {
				t.Fatalf("expected len() 0, got %d", len(results))
			}
		}
	})

	t.Run("reuse results", func(t *testing.T) {
		results := trie.Find(nil, keys[0])
		results2 := trie.Find(results[:0], keys[2])
		if len(results) != len(results2) {
			t.Fatalf("expected len() %d, got %d", len(results), len(results2))
		}
		if &results[0] != &results2[0] {
			t.Fatal("expected results slice to be reused")
		}

		results = make([]int, 1)
		results2 = trie.Find(results, keys[1])
		if len(results2) != 2 {
			t.Fatalf("expected len() %d, got %d", 2, len(results2))
		}
		if &results[0] == &results2[0] {
			t.Fatal("expected results slice to be appended")
		}
	})

	t.Run("partial match", func(t *testing.T) {
		var trie prefixtrie.Trie
		trie.Add("1234567", 1)
		results := trie.Find(nil, "1245")
		if len(results) != 0 {
			t.Fatalf("expected len() %d, got %d", 0, len(results))
		}
	})
}

func Example() {
	var trie prefixtrie.Trie
	trie.Add("www.google.com", 0)
	trie.Add("www.foogle.net", 1)
	fmt.Println(
		trie.Find(nil, "www.google.com"),
		trie.Find(nil, "www.foogle.net"),
		trie.Find(nil, "www"),
		trie.Find(nil, "ogle"),
		trie.Find(nil, "fo"),
		trie.Find(nil, "google.com"),
	)
	// Output: [0] [1] [1 0] [0 1] [1] [0]
}

func Benchmark(b *testing.B) {
	var trie prefixtrie.Trie
	rand.Seed(1)
	words := make([]string, 0, 1000)
	for cap(words) > len(words) {
		words = append(words, randomHex())
	}
	for i, word := range words {
		trie.Add(word, i*25)
	}
	corpus := strings.Join(words, " ")

	b.Run("brute force", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			_ = strings.Index(corpus, words[n%len(words)])
		}
	})

	b.Run("prefix trie", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			_ = trie.Find(nil, words[n%len(words)])
		}
	})

	b.Run("prefix trie with cached results", func(b *testing.B) {
		b.ReportAllocs()
		v := []int(nil)
		for n := 0; n < b.N; n++ {
			v = trie.Find(v[:0], words[n%len(words)])
		}
	})

	b.Run("parallel", func(b *testing.B) {
		b.Run("brute force", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				n := 0
				for pb.Next() {
					_ = strings.Index(corpus, words[n%len(words)])
					n++
				}
			})
		})

		b.Run("prefix trie with cached results", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				v := []int(nil)
				n := 0
				for pb.Next() {
					v = trie.Find(v[:0], words[n%len(words)])
					n++
				}
			})
		})
	})
}

func randomHex() string {
	buf := make([]byte, 24)
	raw := buf[len(buf)/2:]
	if _, err := rand.Read(raw); err != nil { // nolint: gosec
		panic(err)
	}
	for i, bt := range raw {
		lo, hi := bt&0x0f, (bt&0xf0)>>4
		lo = (lo/10)*('a'-'0') + '0' + (lo % 10)
		hi = (hi/10)*('a'-'0') + '0' + (hi % 10)
		buf[i*2], buf[i*2+1] = lo, hi
	}
	return *(*string)(unsafe.Pointer(&buf)) // nolint: gosec
}

func reverse(str string) string {
	buf := make([]byte, len(str))
	for i, r := range str {
		s := utf8.RuneLen(r)
		utf8.EncodeRune(buf[len(buf)-i-s:], r)
	}
	return *(*string)(unsafe.Pointer(&buf)) // nolint: gosec
}
