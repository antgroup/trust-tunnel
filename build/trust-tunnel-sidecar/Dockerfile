FROM docker.m.daocloud.io/busybox:latest

ADD ./build/trust-tunnel-sidecar/superman.sh /
RUN chmod +x /superman.sh
CMD ["/superman.sh", "root", "/bin/bash"]
