FROM scratch
COPY mostcomm /usr/local/bin/mostcomm
ENTRYPOINT ["/usr/local/bin/mostcomm"]
