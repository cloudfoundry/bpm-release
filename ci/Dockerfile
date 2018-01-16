FROM golang:latest
MAINTAINER PCF Security Enablement <pcf-security-enablement@pivotal.io>

# install libseccomp
# The GPG key added is Paul Moore's PGP key.
RUN \
  gpg --keyserver pgp.mit.edu --recv-keys 0x55E45A5AE8CA7C8A && \
  wget https://github.com/seccomp/libseccomp/releases/download/v2.3.2/libseccomp-2.3.2.tar.gz && \
  wget https://github.com/seccomp/libseccomp/releases/download/v2.3.2/libseccomp-2.3.2.tar.gz.asc && \
  gpg --verify libseccomp-2.3.2.tar.gz.asc libseccomp-2.3.2.tar.gz && \
  tar zxf libseccomp-2.3.2.tar.gz && \
  cd libseccomp-2.3.2/  && \
  ./configure && \
  make && \
  make install

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
