FROM golang:latest
MAINTAINER PCF Security Enablement <pcf-security-enablement@pivotal.io>

# install libseccomp
RUN \
  wget https://github.com/seccomp/libseccomp/releases/download/v2.3.2/libseccomp-2.3.2.tar.gz && \
  tar zxf libseccomp-2.3.2.tar.gz && \
  cd libseccomp-2.3.2/  && \
  ./configure && \
  make && \
  make install

RUN apt-get update && apt-get -y install pkg-config

# install runc
RUN \
  go get -u -v github.com/opencontainers/runc && \
  cd $GOPATH/src/github.com/opencontainers/runc && \
  git checkout v1.0.0-rc3 && \
  make && \
  make install

# # add vcap user for test
RUN \
  groupadd vcap -g 3000 && \
  useradd vcap -u 2000 -g 3000

# # add required bosh directories for test
RUN \
  mkdir -p /var/vcap/packages/runc/bin && \
  mkdir -p /var/vcap/data/packages && \
  mkdir -p /var/vcap/data/crucible && \
  mkdir -p /var/vcap/jobs/

RUN \
  ln -s /usr/local/sbin/runc /var/vcap/packages/runc/bin/runc

RUN \
  chown -R vcap:vcap /var/vcap
