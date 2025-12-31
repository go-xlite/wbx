#!/bin/bash

set -e
# Build all client demos
cd demo/client || exit 1


bun ./src/app_w/index/build.js
bun ./src/app_w/home/build.js
bun ./src/app_w/api-demo/build.js
bun ./src/app_w/sse-test/build.js
bun ./src/app_w/proxy-test/build.js
bun ./src/app_w/ws-test/build.js
bun ./src/app_w/stream-test/build.js

bun ./src/app_g/login/build.js