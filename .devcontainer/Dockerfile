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


USER vscode
RUN curl -fsSL https://mise.run | sh



