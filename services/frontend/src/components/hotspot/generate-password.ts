// Gera uma senha Wi-Fi aleatoria (WPA2, minimo 8 caracteres) para
// pre-preencher o formulario quando WIFI_PASSWORD ainda nao foi
// configurado - o operador ve o valor em texto claro e pode aceitar ou
// trocar antes de salvar.
const PASSWORD_ALPHABET = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789";
const PASSWORD_LENGTH = 12;

export function generateRandomWifiPassword(): string {
  const values = new Uint32Array(PASSWORD_LENGTH);
  crypto.getRandomValues(values);
  return Array.from(values, (value) => PASSWORD_ALPHABET[value % PASSWORD_ALPHABET.length]).join("");
}
