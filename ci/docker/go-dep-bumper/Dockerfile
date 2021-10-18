FROM golang:1

# install basic utils {
RUN \
  apt-get update \
  && apt-get install -y \
    curl \
    git \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*
# }
