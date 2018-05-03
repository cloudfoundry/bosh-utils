FROM ubuntu:xenial

# install basic utils {
RUN \
  apt-get update \
  && apt-get install -y \
    curl \
    git \
    jq \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*
# }

# golang {
WORKDIR /usr/local

RUN \
  GO_INFO=$(curl 'https://golang.org/dl/?mode=json' | jq '.[0].files[] | select(.os == "linux" and .arch == "amd64")') && \
  GO_TAR="$(echo "$GO_INFO" | jq -r '.filename')" && \
  GO_SHA="$(echo "$GO_INFO" | jq -r '.sha256')" && \
  curl -fSL https://storage.googleapis.com/golang/$GO_TAR -o $GO_TAR \
  && echo $GO_SHA $GO_TAR | sha256sum -c - \
  && tar -xzf $GO_TAR \
  && rm -f $GO_TAR

ENV \
  PATH=/usr/local/go/bin:$PATH \
  GOPATH=/root/go \
  GOROOT=/usr/local/go
# }

# install dep {
RUN \
  tag="$(curl https://github.com/golang/dep/releases/latest | grep --only-matching -E 'v[0-9]+\.[0-9]+\.[0-9]+')" \
  && curl https://github.com/golang/dep/releases/download/$tag/dep-linux-amd64 -o /usr/bin/dep \
  && chmod a+x /usr/bin/dep
# }
