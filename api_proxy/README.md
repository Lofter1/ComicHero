# API Proxy

Used to proxy api calls and bypass CORS restrictions

## Setup

### Environment Variables

PROXY_URL = The full URL including protocol and port
POCKETBASE_URL = Full URL to Pocketbase, needed for Auth requests

METRON_USERNAME = Metron Username for API login
METRON_PASSWORD = Metron Password for API login

GCD_USERNAME = GCD Username for API login
GCD_PASSWORD = GCD Password for API login

### Optional Environment Variables
REDIS_ADDR = Location of the Redis server for caching. If not set, uses in memory cache
