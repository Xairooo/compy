package main

import (
	"flag"
	"fmt"
	"github.com/barnacs/compy/proxy"
	tc "github.com/barnacs/compy/transcoder"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
)

var (
	host  = flag.String("host", ":9999", "<host:port>")
	cert  = flag.String("cert", "", "proxy cert path")
	key   = flag.String("key", "", "proxy cert key path")
	ca    = flag.String("ca", "", "CA path")
	caKey = flag.String("cakey", "", "CA key path")

	brotli = flag.Int("brotli", -1, "Brotli compression level (0-11, default 6)")
	jpeg   = flag.Int("jpeg", 50, "jpeg quality (1-100, 0 to disable)")
	gif    = flag.Bool("gif", true, "transcode gifs into static images")
	gzip   = flag.Int("gzip", -1, "gzip compression level (0-9, default 6)")
	png    = flag.Bool("png", true, "transcode png")
	minify = flag.Bool("minify", false, "minify css/html/js - WARNING: tends to break the web")
)

func main() {
	flag.Parse()

	p := proxy.New()

	if (*ca == "") != (*caKey == "") {
		log.Fatalln("must specify both CA certificate and key")
	}

	if (*cert == "") != (*key == "") {
		log.Fatalln("must specify both certificate and key")
	}
	p.Cert = cert
	p.Key = key

	if *ca != "" {
		if err := p.EnableMitm(*ca, *caKey); err != nil {
			fmt.Println("not using mitm:", err)
		}
	}

	if *jpeg != 0 {
		p.AddTranscoder("image/jpeg", tc.NewJpeg(*jpeg))
	}
	if *gif {
		p.AddTranscoder("image/gif", &tc.Gif{})
	}
	if *png {
		p.AddTranscoder("image/png", &tc.Png{})
	}

	var ttc proxy.Transcoder
	if *minify {
		ttc = &tc.Zip{tc.NewMinifier(), *brotli, *gzip, false}
	} else {
		ttc = &tc.Zip{&tc.Identity{}, *brotli, *gzip, true}
	}

	p.AddTranscoder("text/css", ttc)
	p.AddTranscoder("text/html", ttc)
	p.AddTranscoder("text/javascript", ttc)
	p.AddTranscoder("application/javascript", ttc)
	p.AddTranscoder("application/x-javascript", ttc)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			read := atomic.LoadUint64(&p.ReadCount)
			written := atomic.LoadUint64(&p.WriteCount)
			log.Printf("compy exiting, total transcoded: %d -> %d (%3.1f%%)",
				read, written, float64(written)/float64(read)*100)
			os.Exit(0)
		}
	}()

	log.Printf("compy listening on %s", *host)
	p.Host = *host

	var err error
	if *cert != "" {
		err = p.StartTLS(*host, *cert, *key)
	} else {
		err = p.Start(*host)
	}
	log.Fatalln(err)
}
