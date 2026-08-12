package network

import "testing"

// Saida real do "nmcli -t -f SSID,CHAN,FREQ,SIGNAL device wifi list"
// na maquina do usuario, incluindo os casos que quebram um parser
// ingenuo: SSID vazio (rede oculta), SSID duplicado em canais
// diferentes (repetidor) e SSID contendo ":" escapado pelo nmcli.
const nmcliSample = `Jorge:11:2462 MHz:100
WIFI2.4:10:2457 MHz:97
WIFI2.4b:36:5180 MHz:89
:48:5240 MHz:40
WIFI2.4:1:2412 MHz:55
Casa\:Wifi:6:2437 MHz:70
lixo-sem-canal:x:2437 MHz:10
`

func TestParseWifiScanOrdenaPorSinal(t *testing.T) {
	networks := parseWifiScan(nmcliSample)

	if len(networks) != 4 {
		t.Fatalf("esperava 4 redes (oculta descartada, duplicada colapsada, linha invalida ignorada), veio %d: %+v", len(networks), networks)
	}
	if networks[0].SSID != "Jorge" || networks[0].Channel != 11 {
		t.Errorf("esperava Jorge no canal 11 em primeiro (maior sinal), veio %+v", networks[0])
	}
}

// O canal certo importa mais que tudo aqui: e ele que o hotspot usa
// para subir o AP na mesma frequencia da rede ancora.
func TestParseWifiScanColapsaSSIDDuplicadoMantendoMaiorSinal(t *testing.T) {
	for _, network := range parseWifiScan(nmcliSample) {
		if network.SSID != "WIFI2.4" {
			continue
		}
		if network.Channel != 10 {
			t.Fatalf("WIFI2.4 aparece nos canais 10 (sinal 97) e 1 (sinal 55); esperava o canal 10, veio %d", network.Channel)
		}
		return
	}
	t.Fatal("WIFI2.4 nao apareceu no resultado")
}

func TestParseWifiScanPreservaDoisPontosEscapadoNoSSID(t *testing.T) {
	for _, network := range parseWifiScan(nmcliSample) {
		if network.SSID == "Casa:Wifi" {
			if network.Channel != 6 {
				t.Fatalf("esperava canal 6 para 'Casa:Wifi', veio %d", network.Channel)
			}
			return
		}
	}
	t.Fatal("SSID com ':' escapado foi perdido no parsing")
}

func TestParseWifiScanVazio(t *testing.T) {
	if networks := parseWifiScan(""); len(networks) != 0 {
		t.Fatalf("esperava lista vazia, veio %+v", networks)
	}
}
