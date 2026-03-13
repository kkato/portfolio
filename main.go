package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

//go:embed templates static
var files embed.FS

type App struct {
	Title       string
	Description string
	URL         string
	GitHub      string
	Tech        []string
}

const annotationPrefix = "portfolio.kkato.app/"

var tmpl = template.Must(template.ParseFS(files, "templates/index.html"))

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("failed to get in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create clientset: %v", err)
	}

	http.Handle("/static/", http.FileServer(http.FS(files)))

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
