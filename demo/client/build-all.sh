#!/bin/bash

set -e
# Build all client demos
cd demo/client || exit 1


bun ./src/index/build.js
bun ./src/home/build.js
bun ./src/api-demo/build.js
bun ./src/sse-test/build.js
bun ./src/proxy-test/build.js
bun ./src/ws-test/build.js
bun ./src/stream-test/build.js