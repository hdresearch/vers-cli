package test

import (
	"bytes"
	"io"
	"os"
)

// Call a function and return its output
// from stdout
func GetStdOut(towrap func ()) string {

	old := os.Stdout // keep backup of the real stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

	towrap()

	outC := make(chan string)
    // copy the output in a separate goroutine so printing can't block indefinitely
    go func() {
        var buf bytes.Buffer
        io.Copy(&buf, r)
        outC <- buf.String()
    }()

    // back to normal state
    w.Close()
    os.Stdout = old // restoring the real stdout
    out := <-outC

	return out

}