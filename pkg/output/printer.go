package output

import (
	"fmt"
	"io"
	"os"
)

type Printer struct {
	w io.Writer
}

func New() *Printer {
	return &Printer{w: os.Stdout}
}

func (p *Printer) OK(msg string)      { fmt.Fprintf(p.w, "[OK  ] %s\n", msg) }
func (p *Printer) Fail(msg string)    { fmt.Fprintf(p.w, "[FAIL] %s\n", msg) }
func (p *Printer) Info(msg string)    { fmt.Fprintf(p.w, "[    ] %s\n", msg) }
func (p *Printer) Skip(msg string)    { fmt.Fprintf(p.w, "[SKIP] %s\n", msg) }
func (p *Printer) Section(msg string) { fmt.Fprintf(p.w, "\n=== %s ===\n", msg) }
