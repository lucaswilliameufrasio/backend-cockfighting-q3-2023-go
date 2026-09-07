package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Shared compose stack — sobe uma vez para todos os testes (padrão do Rust)
// ---------------------------------------------------------------------------

var baseURL = func() string {
	if p := os.Getenv("TEST_API_PORT"); p != "" {
		return "http://localhost:" + p
	}
	return "http://localhost:8080"
}()

var composeReady = false

func ensureCompose(t *testing.T) {
	t.Helper()
	if composeReady {
		return
	}
	_ = runCompose("down", "-v")

	if out, err := runComposeOutput("up", "--build", "-d"); err != nil {
		t.Fatalf("docker compose up falhou: %v\n%s", err, out)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 30; i++ {
		resp, err := client.Get(baseURL + "/health-check")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				composeReady = true
				return
			}
			t.Logf("aguardando API... tentativa %d (status %d)", i+1, resp.StatusCode)
		} else {
			t.Logf("aguardando API... tentativa %d (%v)", i+1, err)
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("API nunca ficou pronta")
}

func runCompose(args ...string) error {
	cmd := exec.Command("docker", append([]string{"compose", "-f", "docker-compose.test.yml"}, args...)...)
	return cmd.Run()
}

func runComposeOutput(args ...string) (string, error) {
	cmd := exec.Command("docker", append([]string{"compose", "-f", "docker-compose.test.yml"}, args...)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	return out.String(), cmd.Run()
}

func TestMain(m *testing.M) {
	code := m.Run()
	_, _ = runComposeOutput("down", "-v")
	os.Exit(code)
}

// ---------------------------------------------------------------------------
// Helpers HTTP
// ---------------------------------------------------------------------------

func doGet(t *testing.T, path string) *http.Response {
	t.Helper()
	ensureCompose(t)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + path)
	if err != nil {
		t.Fatalf("GET %s falhou: %v", path, err)
	}
	return resp
}

func doPost(t *testing.T, path, body string) *http.Response {
	t.Helper()
	ensureCompose(t)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(baseURL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s falhou: %v", path, err)
	}
	return resp
}

func uniqueApelido(prefix string) string {
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
}

type personBody struct {
	ID         string   `json:"id"`
	Apelido    string   `json:"apelido"`
	Nome       string   `json:"nome"`
	Nascimento string   `json:"nascimento"`
	Stack      []string `json:"stack"`
}

// ---------------------------------------------------------------------------
// Testes de contrato (espelham benchmarks/contract-ko.js)
// ---------------------------------------------------------------------------

func TestHealthCheck(t *testing.T) {
	resp := doGet(t, "/health-check")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200, veio %d", resp.StatusCode)
	}
}

func TestCreatePerson201(t *testing.T) {
	apelido := uniqueApelido("integ")
	body := fmt.Sprintf(`{"apelido":%q,"nome":"Integration","nascimento":"1990-01-01","stack":["go","postgres"]}`, apelido)
	resp := doPost(t, "/pessoas", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 201, veio %d: %s", resp.StatusCode, b)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "/pessoas/") {
		t.Fatalf("Location inesperado: %q", loc)
	}
	var created personBody
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if created.ID == "" {
		t.Fatal("id ausente na resposta de criação")
	}
}

func TestCreateDuplicateNickname422(t *testing.T) {
	apelido := uniqueApelido("integ")
	body := fmt.Sprintf(`{"apelido":%q,"nome":"Integration","nascimento":"1990-01-01"}`, apelido)
	if resp := doPost(t, "/pessoas", body); resp.StatusCode != http.StatusCreated {
		t.Fatalf("primeira criação: esperava 201, veio %d", resp.StatusCode)
	}
	resp := doPost(t, "/pessoas", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("duplicado: esperava 422, veio %d", resp.StatusCode)
	}
}

func TestCreateInvalidJSON400(t *testing.T) {
	resp := doPost(t, "/pessoas", "not json")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("esperava 400, veio %d", resp.StatusCode)
	}
}

func TestCreateEmptyBody400or422(t *testing.T) {
	resp := doPost(t, "/pessoas", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("esperava 400 ou 422, veio %d", resp.StatusCode)
	}
}

func TestCreateInvalidField422(t *testing.T) {
	resp := doPost(t, "/pessoas", `{"apelido":"","nome":"Bad","nascimento":"1990-01-01"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("esperava 422, veio %d", resp.StatusCode)
	}
}

func TestGetPersonByID200(t *testing.T) {
	apelido := uniqueApelido("integ")
	body := fmt.Sprintf(`{"apelido":%q,"nome":"Integration","nascimento":"1990-01-01","stack":["go"]}`, apelido)
	createResp := doPost(t, "/pessoas", body)
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: esperava 201, veio %d", createResp.StatusCode)
	}
	var created personBody
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("setup: corpo inválido: %v", err)
	}

	resp := doGet(t, "/pessoas/"+created.ID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200, veio %d", resp.StatusCode)
	}
	var got personBody
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if got.ID != created.ID || got.Apelido != apelido || got.Nome != "Integration" || got.Nascimento != "1990-01-01" {
		t.Fatalf("pessoa retornada diverge: %+v", got)
	}
}

func TestGetUnknownID404(t *testing.T) {
	resp := doGet(t, "/pessoas/00000000-0000-0000-0000-000000000000")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("esperava 404, veio %d", resp.StatusCode)
	}
}

func TestSearchFoundAndEmpty(t *testing.T) {
	apelido := uniqueApelido("integsearch")
	body := fmt.Sprintf(`{"apelido":%q,"nome":"Searchable Integration","nascimento":"1990-01-01"}`, apelido)
	if resp := doPost(t, "/pessoas", body); resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: esperava 201, veio %d", resp.StatusCode)
	}

	resp := doGet(t, "/pessoas?t="+apelido)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("hit: esperava 200, veio %d", resp.StatusCode)
	}
	var found []personBody
	if err := json.NewDecoder(resp.Body).Decode(&found); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	hit := false
	for _, p := range found {
		if p.Apelido == apelido {
			hit = true
		}
	}
	if !hit {
		t.Fatalf("busca não achou %q: %+v", apelido, found)
	}

	emptyResp := doGet(t, "/pessoas?t=nohit_"+apelido)
	defer emptyResp.Body.Close()
	if emptyResp.StatusCode != http.StatusOK {
		t.Fatalf("miss: esperava 200, veio %d", emptyResp.StatusCode)
	}
}

func TestSearchMissingTerm400(t *testing.T) {
	resp := doGet(t, "/pessoas")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("esperava 400, veio %d", resp.StatusCode)
	}
}

func TestCount200PlainText(t *testing.T) {
	resp := doGet(t, "/contagem-pessoas")
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		t.Fatalf("esperava 2XX, veio %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	s := strings.TrimSpace(string(b))
	if s == "" {
		t.Fatal("corpo vazio na contagem")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			t.Fatalf("contagem deveria ser texto puro numérico, veio %q", s)
		}
	}
}
