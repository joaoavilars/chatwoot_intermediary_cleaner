package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// cleanContent limpa o campo "content" do payload removendo espaços e quebras de linha extras
func cleanContent(data map[string]interface{}) {
	if content, ok := data["content"].(string); ok {
		data["content"] = strings.TrimSpace(content)
	}
}

func makeHandler(evolutionURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Erro ao ler body: %v", err)
			http.Error(w, "Erro ao ler corpo", http.StatusBadRequest)
			return
		}

		// Tenta fazer parse do JSON e limpar o campo content
		outBody := body
		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err == nil {
			original, _ := data["content"].(string)
			cleanContent(data)
			cleaned, _ := data["content"].(string)

			if original != cleaned {
				log.Printf("Content limpo: %q → %q", original, cleaned)
			}

			if outBody, err = json.Marshal(data); err != nil {
				log.Printf("Erro ao serializar JSON: %v", err)
				http.Error(w, "Erro interno", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("Body não é JSON válido, encaminhando sem modificação")
		}

		// Encaminha para Evolution API preservando o mesmo path/query
		targetURL := evolutionURL + r.URL.RequestURI()
		req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(outBody))
		if err != nil {
			log.Printf("Erro ao criar request: %v", err)
			http.Error(w, "Erro interno", http.StatusInternalServerError)
			return
		}

		// Copia todos os headers (Content-Length é recalculado automaticamente)
		for key, values := range r.Header {
			if strings.EqualFold(key, "content-length") {
				continue
			}
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("Erro ao encaminhar para Evolution API (%s): %v", targetURL, err)
			http.Error(w, "Erro ao encaminhar requisição", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Copia headers da resposta
		for key, values := range resp.Header {
			for _, v := range values {
				w.Header().Add(key, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)

		log.Printf("[%d] %s → %s", resp.StatusCode, r.URL.Path, targetURL)
	}
}

func main() {
	evolutionURL := os.Getenv("EVOLUTION_API_URL")
	if evolutionURL == "" {
		evolutionURL = "http://evolution_api:8080"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	http.HandleFunc("/", makeHandler(evolutionURL))

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Intermediário rodando em %s → Evolution API: %s", addr, evolutionURL)
	log.Fatal(http.ListenAndServe(addr, nil))
}
