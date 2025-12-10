FROM scratch

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/socks-server /usr/local/bin/socks-server

ENTRYPOINT ["/usr/local/bin/socks-server"]