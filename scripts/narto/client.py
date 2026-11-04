"""[narto] Cliente limitado das APIs existentes; não toca Docker nem volumes."""
import http.cookiejar
import json
import urllib.error
import urllib.parse
import urllib.request


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise RuntimeError('[narto] Redirecionamento inesperado da API')


class API:
    def __init__(self, base, headers=None):
        parsed = urllib.parse.urlsplit(base)
        if parsed.username or parsed.password or parsed.query or parsed.fragment:
            raise ValueError('[narto] URL da API inválida')
        if parsed.scheme != 'https' and not (parsed.scheme == 'http' and parsed.hostname in ('127.0.0.1', 'localhost', '::1')):
            raise ValueError('[narto] API exige HTTPS ou loopback')
        self.base = base.rstrip('/')
        self.headers = headers or {}
        self.opener = urllib.request.build_opener(NoRedirect(), urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def call(self, path, method='GET', data=None, missing=False):
        payload = None if data is None else json.dumps(data).encode()
        request = urllib.request.Request(self.base + path, payload, method=method,
            headers={'Content-Type': 'application/json', **self.headers})
        try:
            with self.opener.open(request, timeout=30) as response:
                body = response.read(2 * 1024 * 1024 + 1)
                if len(body) > 2 * 1024 * 1024:
                    raise RuntimeError('[narto] Resposta da API demasiado grande')
                return json.loads(body) if body else None
        except urllib.error.HTTPError as error:
            if missing and error.code == 404:
                return None
            raise RuntimeError(f'[narto] API recusou {method} {path}: HTTP {error.code}') from None
        except urllib.error.URLError:
            raise RuntimeError('[narto] Serviço indisponível; inicia o Bindnet antes de aplicar') from None
