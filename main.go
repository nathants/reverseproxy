package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

var (
	timeout   time.Duration
	upstreams = make(map[string]string)
)

func handler(ctx *fasthttp.RequestCtx) {
	c := &fasthttp.Client{Dial: fasthttpproxy.FasthttpProxyHTTPDialerTimeout(timeout)}
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
		fmt.Printf("err: no upstream available for host: %s\n", host)
		return
	}
	ctx.Request.CopyTo(req)
	req.SetHost(upstream)
	req.URI().SetScheme("http")
	err := c.DoTimeout(req, res, timeout)
	if err != nil {
		ctx.SetStatusCode(500)
		fmt.Printf("err: upstream did not respond: %s\n", err)
		return
	}
	res.CopyTo(&ctx.Response)
	fmt.Println(res.StatusCode(), string(req.Header.Method()), req.URI())
}

type args struct {
	Addr           string   `arg:"-a,--addr" default:":443"`
	TimeoutSeconds int      `arg:"-t,--timeout" default:"5"`
	SSLCert        string   `arg:"-c,--ssl-cert"`
	SSLKey         string   `arg:"-k,--ssl-key"`
	Upstream       []string `arg:"-u,--upstream" help:"may specify multiple times. --upstream example.com=localhost:8080"`
}

func (*args) Description() string {
	return "\na reverse proxy for one or more http upstreams behind a single wildcard certificate\n"
}

func main() {
	a := args{}
	arg.MustParse(&a)
	timeout = time.Duration(a.TimeoutSeconds) * time.Second
	for _, upstreamArg := range a.Upstream {
		parts := strings.SplitN(upstreamArg, "=", 2)
		domain := parts[0]
		upstream := parts[1]
		upstreams[domain] = upstream
	}
	var err error
	if a.SSLKey != "" && a.SSLCert != "" {
		fmt.Println("serve tls:", a.SSLKey, a.SSLCert, a.Addr)
		err = fasthttp.ListenAndServeTLS(a.Addr, a.SSLCert, a.SSLKey, handler)
	} else {
		fmt.Println("serve:", a.Addr)
		err = fasthttp.ListenAndServe(a.Addr, handler)
	}
	if err != nil {
		panic(err)
	}
}
