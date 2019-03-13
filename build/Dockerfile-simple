FROM scratch

COPY bin/goldpinger /goldpinger
COPY static /static

ENTRYPOINT ["/goldpinger", "--static-file-path", "/static"]
