FROM ghcr.io/cloudfoundry/bosh/golang-release
MAINTAINER CF BPM <cf-bpm@pivotal.io>

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y \
        dnsutils \
        netcat-openbsd \
        pkg-config \
        strace \
        vim-nox

# add required bosh directories for test
RUN mkdir -p /var/vcap/packages/bpm/bin \
    && mkdir -p /var/vcap/data/packages \
    && mkdir -p /var/vcap/data/bpm \
    && mkdir -p /var/vcap/jobs \
    && mkdir -p /var/vcap/store

# copy runc, tinit binaries to /bin
ADD runc-linux/runc.amd64 /bin/runc
ADD tini/tini-amd64 /bin/tini
RUN ln -s /bin/runc /var/vcap/packages/bpm/bin/runc \
    && chmod +x /var/vcap/packages/bpm/bin/runc \
    && ln -s /bin/tini /var/vcap/packages/bpm/bin/tini \
    && chmod +x /var/vcap/packages/bpm/bin/tini

# add vcap user for tests; grant ownership of /var/vcap
RUN groupadd vcap -g 3000 \
  && useradd vcap -u 2000 -g 3000 \
  && chown -R vcap:vcap /var/vcap

WORKDIR /bpm

# prevent git error: "fatal: detected dubious ownership in repository at '/bpm'"
RUN git config --global --add safe.directory /bpm