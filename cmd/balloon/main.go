// Command balloon drives the drawing-ballooning pipeline from the terminal.
//
// It exists mainly so the engine can be exercised, demonstrated and debugged
// without the browser frontend in the way.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/dhwanikher/balloon/internal/dimension"
	"github.com/dhwanikher/balloon/internal/layout"
	"github.com/dhwanikher/balloon/internal/model"
	"github.com/dhwanikher/balloon/internal/render"
)

const usage = `balloon - engineering drawing ballooning and inspection reports

usage:
  balloon parse <callout>...      parse drawing callouts and print them as JSON
  balloon demo [-o out.svg]       render a synthetic ballooned drawing
  balloon build -i texts.json [-o out.svg]
                                  run the pipeline over extracted PDF text

examples:
  balloon parse '4X ⌀12.50 ±0.05' '25 H7' '⌖|⌀0.2Ⓜ|A|B|C'
  balloon demo -o demo.svg
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "parse":
		err = cmdParse(os.Args[2:])
	case "demo":
		err = cmdDemo(os.Args[2:])
	case "build":
		err = cmdBuild(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "balloon: unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "balloon:", err)
		os.Exit(1)
	}
}

func cmdParse(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("parse needs at least one callout")
	}
	opt := dimension.DefaultOptions()
	out := make([]dimension.Characteristic, 0, len(args))
	for _, a := range args {
		out = append(out, dimension.Parse(a, opt))
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func cmdDemo(args []string) error {
	fs := flag.NewFlagSet("demo", flag.ExitOnError)
	outPath := fs.String("o", "demo.svg", "output SVG path")
	debug := fs.Bool("debug", false, "draw text boxes and obstacle regions")
	if err := fs.Parse(args); err != nil {
		return err
	}

	d, texts := demoDrawing()
	model.Build(d, texts)

	f, err := os.Create(*outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	st := render.DefaultStyle()
	st.ShowTextBoxes = *debug
	st.ShowObstacles = *debug
	if err := render.SVG(f, d, 0, st); err != nil {
		return err
	}

	report(d, *outPath)
	return nil
}

func cmdBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	in := fs.String("i", "", "JSON file of extracted text items (required)")
	outPath := fs.String("o", "out.svg", "output SVG path")
	debug := fs.Bool("debug", false, "draw text boxes and obstacle regions")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *in == "" {
		return fmt.Errorf("build needs -i")
	}

	raw, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	var payload struct {
		Drawing model.Drawing    `json:"drawing"`
		Texts   []model.TextItem `json:"texts"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("parsing %s: %w", *in, err)
	}

	d := &payload.Drawing
	model.Build(d, payload.Texts)

	f, err := os.Create(*outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	st := render.DefaultStyle()
	st.ShowTextBoxes = *debug
	st.ShowObstacles = *debug
	if err := render.SVG(f, d, 0, st); err != nil {
		return err
	}

	report(d, *outPath)
	return nil
}

// report prints a short summary to stderr so the command is useful when its
// stdout is being redirected.
func report(d *model.Drawing, outPath string) {
	inspectable := len(d.Inspectable())
	unclean := 0
	for _, it := range d.Items {
		if !it.Clean {
			unclean++
		}
	}

	fmt.Fprintf(os.Stderr, "%s: %d characteristics, %d inspectable, %d need attention\n",
		outPath, len(d.Items), inspectable, unclean)

	if w := d.Warnings(); len(w) > 0 {
		fmt.Fprintln(os.Stderr, "\nwarnings:")
		for _, s := range w {
			fmt.Fprintln(os.Stderr, "  -", s)
		}
	}
}

// demoDrawing builds a synthetic part drawing that exercises most of the parser
// and puts enough callouts near each other to make the solver work for its
// placements. It takes no input files so `balloon demo` runs on a fresh clone.
func demoDrawing() (*model.Drawing, []model.TextItem) {
	const w, h = 842, 595

	d := &model.Drawing{
		ID:         "demo",
		Name:       "Bracket, Mounting",
		PartNumber: "DWG-1042",
		Revision:   "C",
		Options:    dimension.DefaultOptions(),
		Pages: []model.Page{{
			Index: 0, Width: w, Height: h,
			Obstacles: []layout.Rect{
				{X: 250, Y: 150, W: 330, H: 240}, // the part view itself
				{X: 560, Y: 470, W: 270, H: 115}, // title block
			},
		}},
	}

	callouts := []struct {
		text string
		x, y float64
	}{
		{"4X ⌀12.50 ±0.05", 120, 120},
		{"⌀25.00 +0.05/-0.02", 120, 180},
		{"R8.0", 120, 240},
		{"60.00 MAX", 120, 300},
		{"25 H7", 120, 360},
		{"M6x1.0-6H", 120, 420},
		{"2 X 45°", 640, 120},
		{"Ra 1.6", 640, 180},
		{"120.45-120.55", 620, 240},
		{"⌖|⌀0.2Ⓜ|A|B|C", 620, 300},
		{"⊥ 0.05 A", 640, 360},
		{"(85.00)", 640, 420},
		{"[42.00]", 300, 430},
		{"15.000", 380, 430},
		{"90°±0.5°", 460, 430},
		{"SEE NOTE 3", 300, 100},
	}

	texts := make([]model.TextItem, 0, len(callouts))
	for _, c := range callouts {
		texts = append(texts, model.TextItem{
			Text: c.text,
			Page: 0,
			Box: layout.Rect{
				X: c.x, Y: c.y,
				W: float64(len([]rune(c.text))) * 5.2,
				H: 10,
			},
		})
	}
	return d, texts
}
