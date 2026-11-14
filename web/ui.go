package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var distEmbedFS embed.FS

// UIIndex 返回前端单页应用入口
func UIIndex(w http.ResponseWriter, r *http.Request) {
	uiFS, err := fs.Sub(distEmbedFS, "dist")
	if err != nil {
		http.Error(w, "UI 未构建", http.StatusInternalServerError)
		return
	}
	data, err := fs.ReadFile(uiFS, "index.html")
	if err != nil {
		http.Error(w, "UI 未构建，请先执行 npm run build", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

// UIAssets 返回前端构建产物
func UIAssets(w http.ResponseWriter, r *http.Request) {
	assetsFS, err := fs.Sub(distEmbedFS, "dist/assets")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))).ServeHTTP(w, r)
}
