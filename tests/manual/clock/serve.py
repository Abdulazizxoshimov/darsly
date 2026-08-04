"""Sinov sahifalari uchun kichik server (kechikish o'lchovi + etalon slayd).

Ishga tushirish:
    python3 tests/manual/clock/serve.py          # 0.0.0.0:8099

Emulyatordan xostga murojaat: http://10.0.2.2:8099/
    /            → kechikish o'lchagichi (soat)
    /slide.html  → yozuv sifati uchun etalon slayd
    /time        → xostning joriy vaqti (ms) — sahifa soatni shunga sinxronlaydi

Nega `/time` kerak: qurilma va xost soatlari orasidagi noma'lum farq
kechikish o'lchoviga to'g'ridan-to'g'ri qo'shilib ketardi. `adb shell date`
bilan aniqlik ±53 ms, HTTP bilan ±9 ms.
"""
import http.server
import os
import socketserver
import time

os.chdir(os.path.dirname(os.path.abspath(__file__)))
PORT = int(os.environ.get("PORT", "8099"))


class Handler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith("/time"):
            body = str(int(time.time() * 1000)).encode()
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(body)
            return
        super().do_GET()

    def log_message(self, *args):
        pass


socketserver.ThreadingTCPServer.allow_reuse_address = True
print(f"tests/manual/clock → http://0.0.0.0:{PORT}/  (emulyatordan: http://10.0.2.2:{PORT}/)")
with socketserver.ThreadingTCPServer(("0.0.0.0", PORT), Handler) as srv:
    srv.serve_forever()
