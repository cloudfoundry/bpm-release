FROM golang:latest
MAINTAINER CF BPM <cf-bpm@pivotal.io>

RUN apt-get update && apt-get -y install pkg-config strace dnsutils vim-nox netcat-openbsd

# copy runc binary to /bin
ADD runc-linux/runc.amd64 /bin/runc
RUN chmod +x /bin/runc

# add vcap user for test
RUN \
  groupadd vcap -g 3000 && \
  useradd vcap -u 2000 -g 3000

# add required bosh directories for test
RUN \
  mkdir -p /var/vcap/packages/bpm/bin && \
  mkdir -p /var/vcap/data/packages && \
  mkdir -p /var/vcap/data/bpm && \
  mkdir -p /var/vcap/jobs/ && \
  mkdir -p /var/vcap/store/

RUN \
  ln -s /bin/runc /var/vcap/packages/bpm/bin/runc

RUN \
  chown -R vcap:vcap /var/vcap

WORKDIR /bpm
