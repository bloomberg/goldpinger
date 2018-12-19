FROM scratch

ADD bin/goldpinger /goldpinger 
COPY static/index.html /static/index.html

ENTRYPOINT ["/goldpinger", "--static-file-path", "/static"]
