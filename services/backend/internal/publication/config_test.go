package publication

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	request := Request{Name: "site", Domains: []string{"example.test", "www.example.test"}, Alias: "web.service.wnet", Port: 8080, SSL: "letsencrypt", ForceHTTPS: true}
	config, err := Render(request, "/etc/nginx/ssl/site/fullchain.pem", "/etc/nginx/ssl/site/privkey.pem")
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"server_name example.test www.example.test;", "set $upstream_frontend http://web.service.wnet:8080;", "resolver 127.0.0.11 valid=10s ipv6=off;", "proxy_pass $upstream_frontend;", "proxy_pass http://127.0.0.1:9180;", "return 308 https://$host$request_uri;"} {
		if !strings.Contains(config, text) {
			t.Errorf("falta %q", text)
		}
	}
}

func TestRejectInjection(t *testing.T) {
	base := Request{Name: "site", Domains: []string{"example.test"}, Alias: "web.wnet", Port: 80, SSL: "none"}
	cases := []Request{base, base, base, base}
	cases[0].Alias = "web;return 200"
	cases[1].Domains = []string{"example.test;include evil"}
	cases[2].Name = "../site"
	cases[3].ForceHTTPS = true
	for _, req := range cases {
		if _, err := Render(req, "", ""); err == nil {
			t.Fatalf("requisição aceite: %+v", req)
		}
	}
}

func TestModes(t *testing.T) {
	for _, mode := range []string{"none", "bindnet", "selfsigned", "letsencrypt"} {
		t.Run(mode, func(t *testing.T) {
			req := Request{Name: "web", Domains: []string{"one.test", "two.test"}, Alias: "web.wnet", Port: 8080, SSL: mode}
			config, err := Render(req, "/etc/nginx/ssl/web/server.crt", "/etc/nginx/ssl/web/server.key")
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(config, "listen 443 ssl") != (mode != "none") {
				t.Fatal("bloco HTTPS incorreto")
			}
			if strings.Contains(config, "127.0.0.1:9180") != (mode == "letsencrypt") {
				t.Fatal("bloco ACME incorreto")
			}
			if strings.Contains(config, "proxy_pass http://web.wnet") {
				t.Fatal("upstream nao deve ser referenciado diretamente")
			}
		})
	}
}
