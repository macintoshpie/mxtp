#! /bin/bash
# Starts a a mock lambda server allowing you to make requests
#
# For example:
# curl http://localhost:8000/.netlify/functions/jockey/leagues/123
set -e

make build

docker rm -f lambda_service 2>&1 >/dev/null || true
docker run -d --rm \
    --name lambda_service \
    -p 9001:9001 \
    -e DOCKER_LAMBDA_STAY_OPEN=1 \
    --env-file .env \
    -v "$PWD":/var/task:ro,delegated \
    lambci/lambda:go1.x ./bin/functions/jockey

# start a proxy server that handles translating to and from APIGateway request/responses
python -c '
from http.server import BaseHTTPRequestHandler
from http.client import parse_headers
import socketserver
from urllib.request import urlopen
from json import dumps, loads
import os
import time

PORT = 8000
LAMBDA_PORT = int(os.getenv("LAMBDA_PORT", "9001"))

class Proxy(BaseHTTPRequestHandler):
    lambda_endpoint = f"http://localhost:{LAMBDA_PORT}/2015-03-31/functions/jockey/invocations"
    def proxy_it(self):
        content_length = self.headers["Content-Length"]
        data_string = ""
        if content_length:
            data_string = self.rfile.read(int(content_length)).decode()
        response = urlopen(self.lambda_endpoint, dumps({
            "path": self.path,
            "httpMethod": self.command,
            "body": data_string,
            "headers": {k: self.headers[k] for k in self.headers.keys()}
        }).encode())

        body = response.read().decode()
        http_response = loads(body)

        headers = http_response.get("headers", {})
        body = http_response["body"] if http_response.get("body") else ""
        status_code = http_response.get("statusCode", 500)
        self.send_response(status_code)
        for header, value in headers.items():
            self.send_header(header, value)
        #self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(bytes(body, "utf-8"))

    def do_GET(self):
        self.proxy_it()

    def do_POST(self):
        self.proxy_it()

    def do_OPTIONS(self):
        self.proxy_it()

started = False
while not started:
    try:
        with socketserver.TCPServer(("", PORT), Proxy) as httpd:
            started = True
            print(f"Proxying from port {PORT} to {LAMBDA_PORT}")
            httpd.serve_forever()
    except:
        print("Port still occupied, waiting...")
        time.sleep(5)
'