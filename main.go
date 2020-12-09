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

var args struct {
	Addr           string   `arg:"-a,--addr" default:":443"`
	TimeoutSeconds int      `arg:"-t,--timeout" default:"5"`
	SSLCert        string   `arg:"-c,--ssl-cert"`
	SSLKey         string   `arg:"-k,--ssl-key"`
	Upstream       []string `arg:"-u,--upstream" help:"may specify multiple times. --upstream example.com=localhost:8080"`
}

func main() {
	arg.MustParse(&args)
	timeout = time.Duration(args.TimeoutSeconds) * time.Second
	for _, upstreamArg := range args.Upstream {
		parts := strings.SplitN(upstreamArg, "=", 2)
		domain := parts[0]
		upstream := parts[1]
		upstreams[domain] = upstream
	}
	var err error
	if args.SSLKey != "" && args.SSLCert != "" {
		fmt.Println("serve tls:", args.SSLKey, args.SSLCert, args.Addr)
		err = fasthttp.ListenAndServeTLS(args.Addr, args.SSLCert, args.SSLKey, handler)
	} else {
		fmt.Println("serve:", args.Addr)
		err = fasthttp.ListenAndServe(args.Addr, handler)
	}
	if err != nil {
		panic(err)
	}
}
