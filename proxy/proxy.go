package proxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync/atomic"
)

type Proxy struct {
	transcoders map[string]Transcoder
	ml          *mitmListener
	ReadCount   uint64
	WriteCount  uint64
	Host        string
	Cert        *string
	Key         *string
}

type Transcoder interface {
	Transcode(*ResponseWriter, *ResponseReader, http.Header) error
}

func New() *Proxy {
	p := &Proxy{
		transcoders: make(map[string]Transcoder),
		ml:          nil,
	}
	return p
}

func (p *Proxy) EnableMitm(ca, key string) error {
	cf, err := newCertFaker(ca, key)
	if err != nil {
		return err
	}
	p.ml = newMitmListener(cf)
	go http.Serve(p.ml, p)
	return nil
}

func (p *Proxy) AddTranscoder(contentType string, transcoder Transcoder) {
	p.transcoders[contentType] = transcoder
}

// TODO: simplify this and following functions with p.Cert and p.Key?
func (p *Proxy) Start(host string) error {
	return http.ListenAndServe(host, p)
}

func (p *Proxy) StartTLS(host, cert, key string) error {
	return http.ListenAndServeTLS(host, cert, key, p)
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("serving request: %s", r.URL)
	if err := p.handle(w, r); err != nil {
		log.Printf("%s while serving request: %s", err, r.URL)
	}
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "CONNECT" {
		return p.handleConnect(w, r)
	}
	// TODO: why does r.URL.Host not include the port?
	if r.URL.Host == "" {
		if r.Method == "GET" && r.URL.Path == "/" {
			// TODO: return usage, version, and link to CA certificate
			read := atomic.LoadUint64(&p.ReadCount)
			written := atomic.LoadUint64(&p.WriteCount)
			io.WriteString(w, fmt.Sprintf("total transcoded: %d -> %d (%3.1f%%)\n",
				read, written, float64(written)/float64(read)*100))
			return nil
		} else if r.Method == "GET" && r.URL.Path == "/cert" {
			if p.Cert == nil {
				w.WriteHeader(http.StatusNotFound)
				return nil
			}

			w.Header().Set("Content-Type", "application/x-x509-ca-cert")
			file, err := os.OpenFile(*p.Cert, os.O_RDONLY, 0)
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err := io.Copy(w, file); err != nil {
				return err
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			return nil
		}
		return nil
	}

	resp, err := forward(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("error forwarding request: %s", err)
	}
	defer resp.Body.Close()
	rw := newResponseWriter(w)
	rr := newResponseReader(resp)
	err = p.proxyResponse(rw, rr, r.Header)
	read := rr.counter.Count()
	written := rw.rw.Count()
	log.Printf("transcoded: %d -> %d (%3.1f%%)", read, written, float64(written)/float64(read)*100)
	atomic.AddUint64(&p.ReadCount, read)
	atomic.AddUint64(&p.WriteCount, written)
	return err
}

func forward(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme == "" {
		if r.TLS != nil && r.TLS.ServerName == r.Host {
			r.URL.Scheme = "https"
		} else {
			r.URL.Scheme = "http"
		}
	}
	if r.URL.Host == "" {
		r.URL.Host = r.Host
	}
	r.RequestURI = ""
	return http.DefaultTransport.RoundTrip(r)
}

func (p *Proxy) proxyResponse(w *ResponseWriter, r *ResponseReader, headers http.Header) error {
	w.takeHeaders(r)
	transcoder, found := p.transcoders[r.ContentType()]
	if !found {
		return w.ReadFrom(r)
	}
	w.setChunked()
	if err := transcoder.Transcode(w, r, headers); err != nil {
		return fmt.Errorf("transcoding error: %s", err)
	}
	return nil
}

func (p *Proxy) handleConnect(w http.ResponseWriter, r *http.Request) error {
	if p.ml == nil {
		return fmt.Errorf("CONNECT received but mitm is not enabled")
	}
	w.WriteHeader(http.StatusOK)
	var conn net.Conn
	if h, ok := w.(http.Hijacker); ok {
		conn, _, _ = h.Hijack()
	} else {
		fw := w.(FlushWriter)
		fw.Flush()
		mconn := newMitmConn(fw, r.Body, r.RemoteAddr)
		conn = mconn
		defer func() {
			<-mconn.closed
		}()
	}
	sconn, err := p.ml.Serve(conn, r.Host)
	if err != nil {
		conn.Close()
		return err
	}
	sconn.Close() // TODO: reuse this connection for https requests
	return nil
}
