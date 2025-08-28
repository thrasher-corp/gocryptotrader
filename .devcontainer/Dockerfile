## Dev Container base image
# See: https://github.com/devcontainers/images/tree/main/src/go
# Use the published "dev-" tagged image for the latest Go version.
# See MCR tags: dev-1.25, dev-1.25-bookworm
ARG VARIANT=dev-1.25-bookworm
FROM mcr.microsoft.com/devcontainers/go:${VARIANT}

USER root
RUN apt-get update \
	&& export DEBIAN_FRONTEND=noninteractive \
	&& apt-get install -y --no-install-recommends \
		git \
		make \
		build-essential \
		bash-completion \
		ca-certificates \
		curl \
		unzip \
		libssl-dev \
		protobuf-compiler \
		jq \
	&& apt-get clean \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

RUN mkdir -p /home/vscode/.cache/go-build /go \
	&& chown -R vscode:vscode /home/vscode/.cache /go

USER vscode