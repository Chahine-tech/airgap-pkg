package output

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type Printer struct {
	mu sync.Mutex
	w  io.Writer
}

func New() *Printer {
	return &Printer{w: os.Stdout}
}

func NewTo(w io.Writer) *Printer {
	return &Printer{w: w}
}

func (p *Printer) OK(msg string)   { p.print("[OK  ] %s\n", msg) }
func (p *Printer) Fail(msg string) { p.print("[FAIL] %s\n", msg) }
func (p *Printer) Info(msg string) { p.print("[    ] %s\n", msg) }
func (p *Printer) Skip(msg string) { p.print("[SKIP] %s\n", msg) }
func (p *Printer) Warn(msg string) { p.print("[WARN] %s\n", msg) }

func (p *Printer) Section(msg string)              { p.print("\n=== %s ===\n", msg) }
func (p *Printer) Custom(prefix, msg string)       { p.print("[%s] %s\n", prefix, msg) }

func (p *Printer) print(format string, args ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, format, args...)
}
