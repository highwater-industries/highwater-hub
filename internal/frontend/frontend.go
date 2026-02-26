package frontend

import (
"embed"
"io/fs"
"net/http"
)

//go:embed all:dist
var distFS embed.FS

// Handler returns an http.Handler that serves the SvelteKit SPA.
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("frontend: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
path := r.URL.Path
if path == "/" {
fileServer.ServeHTTP(w, r)
return
}

f, err := sub.Open(path[1:])
if err != nil {
r.URL.Path = "/"
fileServer.ServeHTTP(w, r)
return
}
f.Close()
		fileServer.ServeHTTP(w, r)
	})
}
