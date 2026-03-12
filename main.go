package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type App struct {
	Title       string
	Description string
	URL         string
	GitHub      string
	Tech        []string
}

const annotationPrefix = "portfolio.kkato.app/"

var tmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>kkato portfolio</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0f172a; color: #e2e8f0; min-height: 100vh; }
    header { padding: 3rem 2rem; text-align: center; border-bottom: 1px solid #1e293b; }
    header h1 { font-size: 2.5rem; font-weight: 700; background: linear-gradient(135deg, #38bdf8, #818cf8); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
    header p { margin-top: 0.5rem; color: #94a3b8; font-size: 1rem; }
    main { max-width: 1100px; margin: 0 auto; padding: 2.5rem 2rem; }
    .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 1.5rem; }
    .card { background: #1e293b; border: 1px solid #334155; border-radius: 12px; padding: 1.5rem; transition: border-color 0.2s, transform 0.2s; }
    .card:hover { border-color: #38bdf8; transform: translateY(-2px); }
    .card h2 { font-size: 1.2rem; font-weight: 600; color: #f1f5f9; }
    .card p { margin-top: 0.5rem; font-size: 0.9rem; color: #94a3b8; line-height: 1.6; }
    .tech { margin-top: 1rem; display: flex; flex-wrap: wrap; gap: 0.4rem; }
    .tech span { background: #0f172a; border: 1px solid #334155; border-radius: 999px; padding: 0.2rem 0.7rem; font-size: 0.75rem; color: #7dd3fc; }
    .links { margin-top: 1.2rem; display: flex; gap: 0.75rem; }
    .links a { font-size: 0.85rem; text-decoration: none; padding: 0.4rem 0.9rem; border-radius: 6px; transition: opacity 0.2s; }
    .links a:hover { opacity: 0.8; }
    .btn-app { background: #0ea5e9; color: #fff; }
    .btn-gh { background: #1e293b; color: #e2e8f0; border: 1px solid #475569; }
    footer { text-align: center; padding: 2rem; color: #475569; font-size: 0.8rem; }
  </style>
</head>
<body>
  <header>
    <h1>kkato.app</h1>
    <p>個人プロジェクト一覧</p>
  </header>
  <main>
    <div class="grid">
      {{range .}}
      <div class="card">
        <h2>{{.Title}}</h2>
        {{if .Description}}<p>{{.Description}}</p>{{end}}
        {{if .Tech}}
        <div class="tech">
          {{range .Tech}}<span>{{.}}</span>{{end}}
        </div>
        {{end}}
        <div class="links">
          <a class="btn-app" href="{{.URL}}" target="_blank" rel="noopener">アプリを開く</a>
          {{if .GitHub}}<a class="btn-gh" href="{{.GitHub}}" target="_blank" rel="noopener">GitHub</a>{{end}}
        </div>
      </div>
      {{end}}
    </div>
  </main>
  <footer>Powered by Kubernetes + Cloudflare Tunnel</footer>
</body>
</html>
`))

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("failed to get in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create clientset: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		apps, err := listApps(clientset)
		if err != nil {
			http.Error(w, "failed to list apps", http.StatusInternalServerError)
			log.Printf("error listing apps: %v", err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, apps); err != nil {
			log.Printf("template error: %v", err)
		}
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func listApps(clientset *kubernetes.Clientset) ([]App, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var apps []App
	for _, ing := range ingresses.Items {
		ann := ing.Annotations

		// visible=false のものはスキップ
		if ann[annotationPrefix+"visible"] == "false" {
			continue
		}
		// portfolio.kkato.app/title アノテーションがないものはスキップ
		title := ann[annotationPrefix+"title"]
		if title == "" {
			continue
		}

		var url string
		if len(ing.Spec.Rules) > 0 {
			url = "https://" + ing.Spec.Rules[0].Host
		}

		var tech []string
		if t := ann[annotationPrefix+"tech"]; t != "" {
			for _, s := range strings.Split(t, ",") {
				tech = append(tech, strings.TrimSpace(s))
			}
		}

		apps = append(apps, App{
			Title:       title,
			Description: ann[annotationPrefix+"description"],
			URL:         url,
			GitHub:      ann[annotationPrefix+"github"],
			Tech:        tech,
		})
	}
	return apps, nil
}
