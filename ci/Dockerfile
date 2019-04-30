FROM golang:latest
MAINTAINER CF BPM <cf-bpm@pivotal.io>

RUN apt-get update && apt-get -y install \
    dnsutils \
    netcat-openbsd \
    pkg-config \
    strace \
    vim-nox \
  && rm -rf /var/lib/apt/lists/*

# add required bosh directories for test
RUN \
  mkdir -p /var/vcap/packages/bpm/bin && \
  mkdir -p /var/vcap/data/packages && \
  mkdir -p /var/vcap/data/bpm && \
  mkdir -p /var/vcap/jobs/ && \
  mkdir -p /var/vcap/store/

# copy runc binary to /bin
ADD runc-linux/runc.amd64 /bin/runc
RUN ln -s /bin/runc /var/vcap/packages/bpm/bin/runc
RUN chmod +x /var/vcap/packages/bpm/bin/runc

# copy tini binary to /bin
ADD tini/tini-amd64 /bin/tini
RUN ln -s /bin/tini /var/vcap/packages/bpm/bin/tini
RUN chmod +x /var/vcap/packages/bpm/bin/tini

# add vcap user for test
RUN \
  groupadd vcap -g 3000 && \
  useradd vcap -u 2000 -g 3000

RUN chown -R vcap:vcap /var/vcap

WORKDIR /bpm
