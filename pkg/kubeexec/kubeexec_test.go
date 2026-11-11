package kubeexec

import (
	"bytes"
	"testing"
)

func TestLimitedBufferKeepsTheBeginning(t *testing.T) {
	var b limitedBuffer
	chunk := bytes.Repeat([]byte("a"), MaxOutput-1)
	for _, p := range [][]byte{chunk, []byte("bc"), []byte("d")} {
		if n, err := b.Write(p); n != len(p) || err != nil {
			t.Fatalf("Write = %d, %v", n, err)
		}
	}
	if b.Len() != MaxOutput || !bytes.HasSuffix(b.Bytes(), []byte("ab")) {
		t.Fatalf("len=%d suffix=%q", b.Len(), b.Bytes()[b.Len()-2:])
	}
}
