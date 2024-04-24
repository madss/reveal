package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	//go:embed "slides.template"
	defaultTemplate []byte
	//go:embed "reveal.js"
	revealFS embed.FS
)

func main() {
	var flags Flags
	flags.Parse()

	var (
		root      string
		filenames []string
	)

	// Verify that the files exists
	for _, file := range flags.Files {
		if stat, err := os.Stat(file); err != nil || stat.IsDir() {
			fmt.Fprintf(os.Stderr, "Error accessing file %s\n", file)
			os.Exit(1)
		}
		d, fn := filepath.Split(file)
		if root != "" && d != root {
			fmt.Fprintln(os.Stderr, "Error: presentation files must be in the same directory")
			os.Exit(1)
		}
		root = d
		filenames = append(filenames, fn)
	}

	// Prepare the title
	title := flags.Title
	if title == "" {
		title = join(filenames)
	}

	// Find the template
	tmplSrc := defaultTemplate
	if flags.Template != "" {
		var err error
		tmplSrc, err = os.ReadFile(flags.Template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading custom template: %s\n", err.Error())
			os.Exit(1)
		}
	}

	// Prepare the template
	tmpl, err := template.New("").Parse(string(tmplSrc))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing template: %s\n", err.Error())
		os.Exit(1)
	}

	// Create the handlers
	contentServer := http.FileServer(http.Dir(root))
	http.Handle("/reveal.js/", http.FileServer(http.FS(revealFS)))
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			contentServer.ServeHTTP(w, req)
			return
		}
		tmpl.Execute(w, map[string]any{
			"Title":      title,
			"Filenames":  filenames,
			"Theme":      flags.Theme,
			"Transition": flags.Transition,
		})
	})

	url := fmt.Sprintf("http://localhost:%d/", flags.Port)

	// Serve the presentation in the background
	go func() {
		fmt.Printf("Revealing presentation on %s\n", url)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", flags.Port), nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting web server: %s\n", err.Error())
			os.Exit(1)
		}
	}()

	// Open the presentation in a browser
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Run()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run()
	case "darwin":
		err = exec.Command("open", url).Run()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s in browser: %s\n", url, err.Error())
		os.Exit(1)
	}

	// Wait for the user to interrupt the process
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
}

type Flags struct {
	Port       int
	Title      string
	Theme      string
	Transition string
	Template   string
	Files      []string
}

func (f *Flags) Parse() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <file>\n", fs.Name())
		fs.PrintDefaults()
	}
	fs.BoolFunc("help", "Print help and exit", func(string) error {
		fs.Usage()
		f.Help(fs.Output())
		return nil
	})
	fs.IntVar(&f.Port, "port", 12345, "The port where the presentation is served")
	fs.StringVar(&f.Title, "title", "", "The title of the presentation")
	fs.StringVar(&f.Theme, "theme", "league", "The theme. See -help for details")
	fs.StringVar(&f.Transition, "transition", "fade", "The transition. See -help for details")
	fs.StringVar(&f.Template, "template", "", "The path to a file containing a custom template")
	fs.Parse(os.Args[1:])
	f.Files = fs.Args()
	if len(f.Files) < 1 {
		flag.Usage()
		os.Exit(2)
	}
}

func (*Flags) Help(w io.Writer) {
	// TODO: Look up names of transitions and themes
	fmt.Fprint(w, `
Available transitions:

	none, fade, slide, convex, concave, and zoom

Available themes:

	beiga, black-contrast, black, blood, dracula, league, moon, night, serif,
	simple, sky, solarized, white-contrast, white, and
	white_contrast_compact_verbatim_headers

Examples of markdown:

Images

	![Title](path)

Tables

	| Foo | Bar | Baz |
	|-----|:---:|----:|
	| 1   |  2  |   3 |
	| 4   |  5  |   6 |

Codes snippets

	`+"```"+`go [6: 2-3]
	func main() {
		fmt.Print("Hello, ")
		fmt.Println("world!")
	}
	`+"```"+`

Fragments

	- Item one
	- Item two <!-- .element: class="fragment" data-fragment-index="1" -->
	- Item three <!-- .element: class="fragment" data-fragment-index="2" -->
`)
	os.Exit(0)
}

func join(s []string) string {
	if len(s) <= 2 {
		return strings.Join(s, " and ")
	}
	return strings.Join(s[:len(s)-2], ", ") + ", and " + s[len(s)-1]
}
