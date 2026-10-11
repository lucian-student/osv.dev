FROM alpine:latest

# 1. Install curl and jq while still root
RUN apk add --no-cache curl jq

# 2. Create a non-root system user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# 3. Set working directory and assign ownership
WORKDIR /app
RUN chown appuser:appgroup /app

# 4. Switch to the non-root user for all subsequent commands and runtime
USER appuser