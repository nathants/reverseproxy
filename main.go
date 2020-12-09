package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/valyala/fasthttp"
)

var (
	timeout   time.Duration
	upstreams = make(map[string]string)
)

var hopHeaders = []string{
	"Connection",          // Connection
	"Proxy-Connection",    // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",          // Keep-Alive
	"Proxy-Authenticate",  // Proxy-Authenticate
	"Proxy-Authorization", // Proxy-Authorization
	"Te",                  // canonicalized version of "TE"
	"Trailer",             // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",   // Transfer-Encoding
	"Upgrade",             // Upgrade
}

func handler(ctx *fasthttp.RequestCtx) {
	c := fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, timeout)
		},
	}
	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()
	host := string(ctx.Host())
	host = strings.SplitN(host, ":", 2)[0]
	upstream, ok := upstreams[host]
	if !ok {
		ctx.SetStatusCode(404)
		fmt.Println("err: no upstream available for host:", host)
		return
	}
	ctx.Request.CopyTo(req)
	req.SetHost(upstream)
	req.URI().SetScheme("http")
	for _, h := range hopHeaders {
		req.Header.Del(h)
	}
	ip, _, err := net.SplitHostPort(ctx.RemoteAddr().String())
	if err == nil {
		req.Header.Add("X-Forwarded-For", ip)
	}
	err = c.DoTimeout(req, res, timeout)
	if err != nil {
		ctx.SetStatusCode(500)
		fmt.Println("err: upstream did not respond before timeout:", req.URI().String(), err)
		return
	}
	for _, h := range hopHeaders {
		res.Header.Del(h)
	}
	res.CopyTo(&ctx.Response)
	fmt.Println(res.StatusCode(), string(req.Header.Method()), req.URI())
}

type args struct {
	Addr           string   `arg:"-a,--addr" default:":443"`
	TimeoutSeconds int      `arg:"-t,--timeout" default:"5"`
	SSLCert        string   `arg:"-c,--ssl-cert"`
	SSLKey         string   `arg:"-k,--ssl-key"`
	Upstream       []string `arg:"-u,--upstream" help:"may specify multiple upstreams: -u a.foo.com=localhost:8080 b.foo.com=localhost:8081"`
	BufferSize     int      `arg:"-b,--buffer" default:"16384" help:"fasthttp.Server.{Read,Write}BufferSize"`
}

func (*args) Description() string {
	return "\nreverse proxy one or more http upstreams behind a wildcard certificate\n"
}

func main() {
	count := 0
	for _, val := range os.Args {
		if val == "-u" || val == "--upstream" {
			count++
		}
	}
	if count > 1 {
		fmt.Println("fatal: do not specify -u multiple times, specify it ones with space seperated values")
		fmt.Println("example: reverseproxy -u foo.com=localhost:8080 bar.com=localhost:8081")
		os.Exit(1)
	}
	a := args{}
	arg.MustParse(&a)
	timeout = time.Duration(a.TimeoutSeconds) * time.Second
	for _, upstreamArg := range a.Upstream {
		parts := strings.SplitN(upstreamArg, "=", 2)
		domain := parts[0]
		upstream := parts[1]
		upstreams[domain] = upstream
		fmt.Printf("upstream: %s => %s\n", domain, upstream)
	}
	s := &fasthttp.Server{
		Handler:         handler,
		ReadBufferSize:  a.BufferSize,
		WriteBufferSize: a.BufferSize,
	}
	var err error
	if a.SSLKey != "" && a.SSLCert != "" {
		fmt.Println("serve tls:", a.SSLKey, a.SSLCert, a.Addr)
		if err != nil {
			panic(err)
		}
		ln, err := net.Listen("tcp4", a.Addr)
		if err != nil {
			panic(err)
		}
		err = s.ServeTLS(ln, a.SSLCert, a.SSLKey)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("serve:", a.Addr)
		ln, err := net.Listen("tcp4", a.Addr)
		if err != nil {
			panic(err)
		}
		err = s.Serve(ln)
		if err != nil {
			panic(err)
		}
	}
	if err != nil {
		panic(err)
	}
}
