"""[narto] Verifica registo idempotente e preservação das zonas existentes."""
import unittest
from provision import append_domain, configuration, same_site
from client import API


class ProvisionTest(unittest.TestCase):
    def test_existing_dns_zones_preserved(self):
        self.assertEqual('costa.bnet,narto.local.com', append_domain('costa.bnet'))
        self.assertEqual('costa.bnet,narto.local.com', append_domain('costa.bnet,narto.local.com'))

    def test_same_site_accepts_nginx_ui_formatting(self):
        self.assertTrue(same_site('# comment\nserver { listen 443; }', 'server{listen 443;}'))
        self.assertFalse(same_site('server_name other;', 'server_name narto.local.com;'))

    def test_upstream_uses_shared_network_and_https(self):
        body = configuration('main')
        self.assertIn('main.frontend.narto.wnet:8080', body)
        self.assertIn('server_name narto.local.com;', body)
        self.assertIn('ssl_certificate /etc/nginx/ssl/narto.local.com/server.crt;', body)
        with self.assertRaises(ValueError):
            configuration('main;evil')

    def test_credentials_cannot_be_redirected_or_sent_plain_to_remote_hosts(self):
        with self.assertRaises(ValueError):
            API('http://remote.example:8090')
        with self.assertRaises(ValueError):
            API('https://user:secret@remote.example')


if __name__ == '__main__':
    unittest.main()
