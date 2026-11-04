#!/usr/bin/env python3
"""[narto] Regista DNS/site/certificado pelas APIs autenticadas do Bindnet/nginx-ui."""
import argparse
from datetime import datetime, timezone, timedelta
import getpass
import json
import os
from pathlib import Path
import re
import sys
from client import API

DOMAIN = 'narto.local.com'
DOMAINS = [DOMAIN, '*.narto.local.com']


def configuration(stage):
    if not re.fullmatch('[a-z0-9-]{1,40}', stage):
        raise ValueError('[narto] Estágio inválido')
    path = Path(__file__).resolve().parents[2] / 'services/gateway/narto-nginx-ui-site.conf'
    return path.read_text().replace('main.frontend.narto.wnet', f'{stage}.frontend.narto.wnet')


def same_site(left, right):
    def normalized(value):
        return re.sub(r'\s+', '', re.sub(r'#.*', '', value))
    return normalized(left) == normalized(right)


def append_domain(value):
    parts = [part for part in re.split(r'[\s,;]+', value) if part]
    if DOMAIN not in parts:
        parts.append(DOMAIN)
    return ','.join(parts)


def ensure_certificate(bindnet):
    certificates = bindnet.call('/api/certificates') or []
    existing = next((item for item in certificates if item['name'] == DOMAIN), None)
    if existing:
        expiry = datetime.fromisoformat(existing['expiresAt'].replace('Z', '+00:00'))
        sans = set(existing.get('dnsNames', []))
        if expiry < datetime.now(timezone.utc) + timedelta(days=7) or not set(DOMAINS).issubset(sans):
            raise RuntimeError('[narto] Certificado existente exige revisão/reemissão no painel Bindnet')
        print('[narto] Certificado existente preservado')
    else:
        bindnet.call('/api/certificates', 'POST', {'name': DOMAIN, 'domains': DOMAINS,
            'validityQuantity': 1, 'validityUnit': 'years'})
        print('[narto] Certificado emitido; verifica a importação na lista do nginx-ui')


def apply(args):
    bindnet = API(args.bindnet)
    # Os segredos só permanecem em memória; não se lê nem altera .env.
    bindnet.call('/api/auth/login', 'POST', {'username': os.getenv('BINDNET_USERNAME') or input('Utilizador Bindnet: '),
        'password': os.getenv('BINDNET_PASSWORD') or getpass.getpass('Senha Bindnet: ')})
    token = os.getenv('NGINX_UI_TOKEN') or getpass.getpass('Token de API nginx-ui: ')
    secure = os.getenv('NGINX_UI_SECURE_SESSION', '')
    nginx = API(args.nginx_ui, {'Authorization': token, 'X-Secure-Session-ID': secure})
    content = configuration(args.stage)
    old = nginx.call('/api/sites/' + DOMAIN, missing=True)
    if old is not None and not same_site(old.get('config', ''), content):
        raise RuntimeError('[narto] Já existe um site diferente; revê-o no nginx-ui antes de substituir')
    ensure_certificate(bindnet)
    if old is None:
        nginx.call('/api/sites/' + DOMAIN, 'POST', {'content': content, 'overwrite': False, 'post_action': ''})
    nginx.call('/api/sites/' + DOMAIN + '/enable', 'POST', {})
    cfg = bindnet.call('/api/dns/config')
    domain_list = append_domain(cfg.get('DOMAINS', ''))
    if domain_list != cfg.get('DOMAINS', ''):
        bindnet.call('/api/dns/config', 'PATCH', {'DOMAINS': domain_list})
    if args.apply_dns:
        bindnet.call('/api/dns/apply', 'POST', {})
    print('[narto] Site HTTPS registado; configuração DNS preservada e acrescida do domínio')
    if not args.apply_dns:
        print('[narto] Aplica DNS pelo painel para atualizar as rotas do resolved; o hotspot não foi alterado')
    print('[narto] Acesso: https://narto.local.com')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--stage', default='main')
    parser.add_argument('--bindnet', default='http://127.0.0.1:8090')
    parser.add_argument('--nginx-ui', default='http://127.0.0.1:9080')
    parser.add_argument('--apply', action='store_true', help='Aplicar pelas APIs autenticadas')
    parser.add_argument('--apply-dns', action='store_true', help='Aplicar também as rotas DNS pelo worker')
    args = parser.parse_args()
    if args.apply:
        apply(args)
    else:
        print(json.dumps({'domain': DOMAIN, 'certificate': {'name': DOMAIN, 'domains': DOMAINS},
            'upstream': f'{args.stage}.frontend.narto.wnet:8080', 'site': configuration(args.stage),
            'dns': 'Acrescentar narto.local.com a DOMAINS, preservando as outras zonas'}, indent=2))


if __name__ == '__main__':
    try:
        main()
    except (RuntimeError, ValueError, KeyError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
