// rd downloads a reel/short from a share URL into the current directory.
//
//	rd https://www.facebook.com/share/r/<code>/
//	rd -o ~/Downloads https://www.facebook.com/reel/<id>
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/alileza/rd/extractor"
)

func main() {
	out := flag.String("o", ".", "output directory, or a file path ending in an extension")
	printURL := flag.Bool("url", false, "print the direct media URL instead of downloading")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: rd [-o dir|file] [-url] <url>\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, flag.Arg(0), *out, *printURL); err != nil {
		fmt.Fprintln(os.Stderr, "rd:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, rawURL, out string, printURL bool) error {
	ex, u, err := extractor.Find(rawURL)
	if err != nil {
		return err
	}

	c := extractor.NewClient()
	m, err := ex.Extract(ctx, c, u)
	if err != nil {
		return fmt.Errorf("%s: %w", ex.Name(), err)
	}
	if printURL {
		fmt.Println(m.URL)
		return nil
	}

	dest := out
	if st, err := os.Stat(out); (err == nil && st.IsDir()) || filepath.Ext(out) == "" {
		dest = filepath.Join(out, m.Filename())
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	if m.Title != "" {
		fmt.Fprintf(os.Stderr, "%s %s (%s): %s\n", m.Site, m.ID, m.Quality, m.Title)
	} else {
		fmt.Fprintf(os.Stderr, "%s %s (%s)\n", m.Site, m.ID, m.Quality)
	}
	if err := download(ctx, m, dest); err != nil {
		return err
	}
	fmt.Println(dest)
	return nil
}

// download streams m.URL into dest via a .part file so an interrupted download
// never leaves a truncated file under the final name.
func download(ctx context.Context, m *extractor.Media, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", extractor.UserAgent)
	for k, v := range m.Headers {
		req.Header.Set(k, v)
	}

	// No client timeout: large files can legitimately take a while. Ctrl-C cancels.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: %s", resp.Status)
	}

	part := dest + ".part"
	f, err := os.Create(part)
	if err != nil {
		return err
	}
	pw := &progress{total: resp.ContentLength, start: time.Now()}
	_, err = io.Copy(io.MultiWriter(f, pw), resp.Body)
	pw.done()
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(part)
		return fmt.Errorf("download: %w", err)
	}
	return os.Rename(part, dest)
}

type progress struct {
	total, n int64
	start    time.Time
	last     time.Time
}

func (p *progress) Write(b []byte) (int, error) {
	p.n += int64(len(b))
	if time.Since(p.last) > 100*time.Millisecond {
		p.last = time.Now()
		p.print()
	}
	return len(b), nil
}

func (p *progress) print() {
	const mb = 1 << 20
	if p.total > 0 {
		fmt.Fprintf(os.Stderr, "\r  %5.1f%%  %.1f / %.1f MB", float64(p.n)*100/float64(p.total), float64(p.n)/mb, float64(p.total)/mb)
	} else {
		fmt.Fprintf(os.Stderr, "\r  %.1f MB", float64(p.n)/mb)
	}
}

func (p *progress) done() {
	p.print()
	fmt.Fprintf(os.Stderr, "  in %s\n", time.Since(p.start).Round(100*time.Millisecond))
}
