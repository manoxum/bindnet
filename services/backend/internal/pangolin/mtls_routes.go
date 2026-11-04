package pangolin

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *service) mtlsCertificates(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := s.managedResource(r, c)
	if err != nil {
		http.Error(w, err.Error(), 403)
		return nil
	}
	path := "/public-resource/" + strconv.Itoa(id) + "/mtls/"
	switch r.Method {
	case "GET":
		// Certificados são públicos, mas limitamos a resposta aos metadados.
		var data struct {
			Certificates []struct {
				ID          int    `json:"mtlsCertificateId"`
				Name        string `json:"name"`
				Fingerprint string `json:"fingerprint"`
			} `json:"certificates"`
		}
		if err = c.call(r.Context(), "GET", path+"certificates", nil, &data); err != nil {
			return err
		}
		writeJSON(w, data)
		return nil
	case "POST":
		var body struct {
			Name        string `json:"name"`
			Certificate string `json:"certificate"`
		}
		if err = decode(w, r, &body); err != nil {
			http.Error(w, err.Error(), 400)
			return nil
		}
		if len(body.Name) > 255 || strings.Contains(body.Certificate, "PRIVATE KEY") {
			http.Error(w, "envie somente o certificado público da CA", 400)
			return nil
		}
		block, _ := pem.Decode([]byte(body.Certificate))
		if block == nil || block.Type != "CERTIFICATE" {
			http.Error(w, "certificado PEM inválido", 400)
			return nil
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil || !cert.IsCA {
			http.Error(w, "informe um certificado público de CA válido", 400)
			return nil
		}
		if err = c.call(r.Context(), "PUT", path+"certificate", body, nil); err != nil {
			return err
		}
	case "DELETE":
		certID, err := strconv.Atoi(r.PathValue("certificateId"))
		if err != nil || certID <= 0 {
			http.Error(w, "certificado inválido", 400)
			return nil
		}
		if err = c.remove(r.Context(), path+"certificate/"+strconv.Itoa(certID)); err != nil {
			return fmt.Errorf("não foi possível remover a CA: %w", err)
		}
	}
	s.record(r, "pangolin_mtls_changed", id)
	w.WriteHeader(204)
	return nil
}
