package network

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// WifiNetwork e uma rede Wi-Fi vista pela placa, com o canal em que ela
// opera - o dado que interessa ao painel para a "rede ancora" do
// hotspot (ver WIFI_ANCHOR_SSID em RULE.md): num radio unico, o AP e a
// associacao de estacao tem que dividir a MESMA frequencia
// ("#channels <= 1"), entao o canal da rede a que o operador quer se
// conectar e o canal em que o AP tem que subir.
type WifiNetwork struct {
	SSID    string `json:"ssid"`
	Channel int    `json:"channel"`
	Freq    int    `json:"freqMhz"`
	Signal  int    `json:"signal"`
}

// RegisterWifiScanRoute expoe a lista de redes Wi-Fi visiveis.
//
// Le o CACHE do NetworkManager (--rescan no) de proposito, nunca
// forcando uma varredura: um scan ativo obriga o radio a saltar de
// canal e, com o AP no ar, interrompe o beacon e chega a derrubar
// clientes conectados - risco documentado em RULE.md para o modo
// AP+STA. O NetworkManager mantem esse cache atualizado sozinho, entao
// o painel recebe dados frescos sem custo nenhum para o hotspot.
func RegisterWifiScanRoute(mux *http.ServeMux) {
	mux.HandleFunc("GET /network/wifi-scan", handleWifiScan)
}

func handleWifiScan(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// -t/-f: saida separada por ":" e sem cabecalho, estavel para
	// parsing (o formato de tabela do nmcli alinha por espacos e muda
	// conforme a largura do terminal).
	output, err := exec.Command("nmcli", "-t", "-f", "SSID,CHAN,FREQ,SIGNAL",
		"device", "wifi", "list", "--rescan", "no").Output()
	if err != nil {
		// Sem NetworkManager acessivel o painel simplesmente nao oferece
		// a lista - nunca e erro fatal, o operador ainda pode digitar o
		// SSID a mao.
		_ = json.NewEncoder(w).Encode([]WifiNetwork{})
		return
	}

	_ = json.NewEncoder(w).Encode(parseWifiScan(string(output)))
}

// parseWifiScan converte a saida "SSID:CHAN:FREQ MHz:SIGNAL" do nmcli.
// Separada do handler para poder ser testada sem NetworkManager.
//
// Redes com o mesmo SSID aparecem uma vez so (a de sinal mais forte):
// varios APs da mesma rede em canais diferentes sao comuns (mesh,
// repetidor - ver "Repeater_359B10" na casa do usuario) e oferecer o
// mesmo nome duas vezes no seletor so confundiria. SSID vazio (rede
// oculta) e descartado: nao da pra ancorar num nome que nao existe.
func parseWifiScan(output string) []WifiNetwork {
	best := map[string]WifiNetwork{}
	for _, line := range strings.Split(output, "\n") {
		// O nmcli escapa ":" dentro do SSID como "\:" - trocar pelo
		// placeholder antes de separar evita quebrar esses nomes.
		const escaped = "\x00"
		fields := strings.Split(strings.ReplaceAll(line, `\:`, escaped), ":")
		if len(fields) < 4 {
			continue
		}
		ssid := strings.TrimSpace(strings.ReplaceAll(fields[0], escaped, ":"))
		if ssid == "" {
			continue
		}
		channel, err := strconv.Atoi(strings.TrimSpace(fields[1]))
		if err != nil {
			continue
		}
		// FREQ vem como "2457 MHz".
		freq, _ := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(fields[2]), " MHz")))
		signal, _ := strconv.Atoi(strings.TrimSpace(fields[3]))

		if current, seen := best[ssid]; seen && current.Signal >= signal {
			continue
		}
		best[ssid] = WifiNetwork{SSID: ssid, Channel: channel, Freq: freq, Signal: signal}
	}

	networks := make([]WifiNetwork, 0, len(best))
	for _, network := range best {
		networks = append(networks, network)
	}
	// Sinal mais forte primeiro: e a ordem util no seletor do painel,
	// porque a rede que o operador quer ancorar e quase sempre a mais
	// proxima. SSID desempata para a ordem ser estavel entre chamadas.
	sort.Slice(networks, func(i, j int) bool {
		if networks[i].Signal != networks[j].Signal {
			return networks[i].Signal > networks[j].Signal
		}
		return networks[i].SSID < networks[j].SSID
	})
	return networks
}
