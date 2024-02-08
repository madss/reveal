package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
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
	slidesTemplate string
	//go:embed "reveal.js"
	revealFS embed.FS
)

func main() {
	var (
		port       int
		theme      string
		transition string
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <file>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.BoolFunc("help", "Print help and exit", func(string) error {
		flag.Usage()
		fmt.Fprint(flag.CommandLine.Output(), `
Available transitions:

    none, fade, slide, convex, concave, and zoom

Available themes:

    beiga, black-contrast, black, blood, dracula, league, moon, night, serif,
    simple, sky, solarized, white-contrast, white, wnd
    white_contrast_compact_verbatim_headers

Examples of markdown:
 
Images:

    ![Title](url)

Tables

    | Foo | Bar | Baz |
    |-----|:---:|----:|
    | 1   | 2   | 3   |
    | 4   | 5   | 6   |

Codes snippets

	`+"```"+`go [6: 2-3]
	func main() {
		fmt.Print("Hello, ")
		fmt.Println("world!")
	}
	`+"```"+`
`)
		os.Exit(0)
		return nil
	})
	flag.IntVar(&port, "port", 12345, "Set the port")
	flag.StringVar(&theme, "theme", "league", "The theme. See -help for details")
	flag.StringVar(&transition, "transition", "fade", "The transition. See -help for details")
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	path := flag.Arg(0)

	// Verify that file exists
	if stat, err := os.Stat(path); err != nil || stat.IsDir() {
		fmt.Fprintf(os.Stderr, "Error accessing file %s\n", path)
		os.Exit(1)
	}
	dir, filename := filepath.Split(path)

	// Load template
	tmpl, err := template.New("").Parse(slidesTemplate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading slides template: %s\n", err.Error())
		os.Exit(1)
	}

	// Create content servers
	revealServer := http.FileServer(http.FS(revealFS))
	contentServer := http.FileServer(http.Dir(dir))

	http.Handle("/reveal.js/", revealServer)
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			contentServer.ServeHTTP(w, req)
			return
		}
		tmpl.Execute(w, map[string]any{
			"Title":      strings.TrimSuffix(filename, ".md"),
			"Filename":   filename,
			"Theme":      theme,
			"Transition": transition,
		})
	})

	go func() {
		fmt.Printf("Revealing %s on http://localhost:%d/\n", filename, port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting web server: %s\n", err.Error())
			os.Exit(1)
		}
	}()

	// Open the presentation in a browser
	url := fmt.Sprintf("http://localhost:%d/", port)
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
		fmt.Fprintf(os.Stderr, "Error opening %s: %s\n", url, err.Error())
		os.Exit(1)
	}

	// Wait for the user to interrupt the process
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
}
