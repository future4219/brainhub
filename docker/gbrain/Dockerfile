FROM oven/bun:1

USER root
RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

USER bun
ENV BUN_INSTALL_BIN=/home/bun/.bun/bin
ENV PATH=/home/bun/.bun/bin:${PATH}

RUN bun install -g github:garrytan/gbrain#v0.45.18.0
RUN test "$(gbrain --version)" = "gbrain 0.45.12.0"

WORKDIR /brain
CMD ["gbrain", "serve", "--http", "--bind", "0.0.0.0", "--port", "3131"]
