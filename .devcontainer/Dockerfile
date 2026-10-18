# Dockerfile for osv.dev development (devcontainer)
FROM mcr.microsoft.com/devcontainers/base:dev-ubuntu26.04

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y --no-install-recommends \
       build-essential \
       curl \
       git \
       protobuf-compiler \
       libprotobuf-dev \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://dl.k6.io/key.gpg | gpg --dearmor -o /usr/share/keyrings/k6-archive-keyring.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | tee /etc/apt/sources.list.d/k6.list && \
    apt-get update && \
    apt-get install -y k6 \
    && rm -rf /var/lib/apt/lists/*

USER vscode
RUN curl -fsSL https://mise.run | sh
ENV PATH="/home/vscode/.local/bin:${PATH}"
RUN mise activate >> ~/.bashrc

