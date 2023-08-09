FROM            alpine:latest
RUN 			apk add --no-cache openssh-client
ENTRYPOINT      ["/bin/sh", "-c"]
CMD             ["/integration/_client.sh"]
COPY            . /integration