FROM bosh/main-ruby-go

# install additional deps {
RUN \
  apt-get update \
  && apt-get install -y \
    zlib1g-dev \
    libsqlite3-dev \
    libpq-dev \
    libmysqlclient-dev \
    lsof \
    psmisc \
    strace \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*
# }
