FROM scratch

WORKDIR /app

COPY socks-server /usr/local/bin/socks-server

ENTRYPOINT ["/usr/local/bin/socks-server"]