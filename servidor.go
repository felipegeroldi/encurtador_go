package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/felipegeroldi/encurtador_go/url"
)

var (
	porta   int
	urlBase string
	stats   chan string
)

// Função especial para inicialização de recursos
func init() {
	porta = 8888
	urlBase = fmt.Sprintf("http://localhost:%d", porta)

	repositorio := url.NovoRepositorioMemoria()
	url.ConfigurarRepositorio(repositorio)
}

func main() {
	stats = make(chan string)
	defer close(stats)
	go registrarEstatisticas(stats)

	http.HandleFunc("/api/encurtar", Encurtador)
	http.HandleFunc("/r/", Redirecionador)
	http.HandleFunc("/api/stats/", Visualizador)

	log.Fatal(http.ListenAndServe(
		fmt.Sprintf(":%d", porta), nil))
}

type Headers map[string]string

// A função precisa seguir a assinatura esperada pelo método HandleFunc
func Encurtador(w http.ResponseWriter, r *http.Request) {
	// Permitir somente req. Post
	if r.Method != "POST" {
		responderCom(w, http.StatusMethodNotAllowed, Headers{
			"Allow": "POST",
		})

		return
	}

	url, nova, err := url.BuscarOuCriarNovaUrl(extrairUrl(r))
	if err != nil {
		responderCom(w, http.StatusBadRequest, nil)
		return
	}

	var status int
	if nova {
		status = http.StatusCreated
	} else {
		status = http.StatusOK
	}

	urlCurta := fmt.Sprintf("%s/r/%s", urlBase, url.Id)
	responderCom(w, status, Headers{
		"Location": urlCurta,
		"Link": fmt.Sprintf("<%s/api/stats/%s>; rel=\"stats\"",
			urlBase, url.Id),
	})
}

func Redirecionador(w http.ResponseWriter, r *http.Request) {
	caminho := strings.Split(r.URL.Path, "/")
	id := caminho[len(caminho)-1]

	if url := url.Buscar(id); url != nil {
		http.Redirect(w, r, url.Destino, http.StatusMovedPermanently)

		stats <- id
	} else {
		http.NotFound(w, r)
	}
}

func registrarEstatisticas(ids <-chan string) {
	for id := range ids {
		url.RegistarClick(id)
		fmt.Printf("Click registrado com sucesso para %s.\n", id)
	}
}

func responderCom(
	w http.ResponseWriter,
	status int,
	headers Headers,
) {
	for k, v := range headers {
		w.Header().Set(k, v)
	}

	w.WriteHeader(status)
}

func extrairUrl(r *http.Request) string {
	url := make([]byte, r.ContentLength)
	r.Body.Read(url)

	return string(url)
}

func Visualizador(w http.ResponseWriter, r *http.Request) {
	caminho := strings.Split(r.URL.Path, "/")
	id := caminho[len(caminho)-1]

	if url := url.Buscar(id); url != nil {
		json, err := json.Marshal(url.Stats())

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		responderComJSON(w, string(json))
	} else {
		http.NotFound(w, r)
	}
}

func responderComJSON(w http.ResponseWriter, resposta string) {
	responderCom(w, http.StatusOK, Headers{
		"Content-Type": "application/json",
	})

	fmt.Fprint(w, resposta)
}
