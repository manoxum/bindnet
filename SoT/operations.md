# Operação do Bindnet

O Bindnet controla recursos físicos da máquina. Subir, parar, reiniciar ou
recriar o Compose pode interromper conectividade, o hotspot e a resolução DNS
para utilizadores ativos. Não execute essas ações sem autorização explícita do
operador.

O ponto de entrada operacional para imagens é `bin/promote`; consulte
[bin/SoT/promote.md](../bin/SoT/promote.md) para o contrato de build,
publicação e promoção. A topologia Compose e as configurações de ambiente são
definidas no repositório raiz.

Antes de testar alterações de rede ou hotspot, prefira validação estática,
testes unitários e revisão de configuração. Teste no host real apenas com
consentimento explícito e janela de manutenção.
