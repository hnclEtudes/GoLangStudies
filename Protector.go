package main

/*
 * A reverse proxy for the server implemented by FiveOnlySrv.go
 * (http://localhost:8181) that protects it by allowing only
 * five requests in and then turning others away with a 503
 * status unavilable error.
 *
 * Basics taken from https://gist.github.com/JalfResi/6287706
 * Thank you, Mr JalfResi!
 */
import(
        "log"
        "net/url"
        "net/http"
        "net/http/httputil"
)

func main() {
        remote, err := url.Parse("http://localhost:8181/")
        if err != nil {
                log.Fatalf("url.Parse of http://localhost:8181/ failed: %s\n", err)
        }

        proxy := httputil.NewSingleHostReverseProxy(remote)
        http.HandleFunc("/", handler(proxy))
        err = http.ListenAndServe(":8080", nil)
        if err != nil {
                panic(err)
        }
}

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
        return func(w http.ResponseWriter, r *http.Request) {
                log.Println(r.URL)
                w.Header().Set("X-Ben", "Rad")
                p.ServeHTTP(w, r)
        }
}
