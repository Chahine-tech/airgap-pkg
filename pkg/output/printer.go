package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("#00d787"))
	styleFail    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f"))
	styleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("#5fafff"))
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd700"))
	styleSkip    = lipgloss.NewStyle().Foreground(lipgloss.Color("#878787"))
	styleSection = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))
	styleMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("#878787"))
	styleAdd     = lipgloss.NewStyle().Foreground(lipgloss.Color("#00d787"))
	styleDel     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f"))
	styleUpd     = lipgloss.NewStyle().Foreground(lipgloss.Color("#5fd7ff"))
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

func (p *Printer) OK(msg string)   { p.println(styleOK.Render("  ✓ ") + " " + msg) }
func (p *Printer) Fail(msg string) { p.println(styleFail.Render("  ✗ ") + " " + msg) }
func (p *Printer) Info(msg string) { p.println(styleInfo.Render("  ● ") + " " + msg) }
func (p *Printer) Warn(msg string) { p.println(styleWarn.Render("  ⚠ ") + " " + msg) }
func (p *Printer) Skip(msg string) { p.println(styleSkip.Render("  ↷ ") + " " + msg) }

func (p *Printer) Section(msg string) {
	bar := strings.Repeat("═", 3)
	line := styleSection.Render(bar + " " + msg + " " + bar)
	p.println("\n" + line + "\n")
}

func (p *Printer) Custom(prefix, msg string) {
	var icon string
	switch prefix {
	case "ADD":
		icon = styleAdd.Render("  + ")
	case "DEL":
		icon = styleDel.Render("  - ")
	case "UPD":
		icon = styleUpd.Render("  ↑ ")
	case "=  ":
		icon = styleMuted.Render("  = ")
	default:
		icon = styleMuted.Render("  " + prefix + " ")
	}
	p.println(icon + " " + msg)
}

func (p *Printer) println(line string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintln(p.w, line)
}
