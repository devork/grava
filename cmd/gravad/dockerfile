FROM alpine:3.6

# ---------------------------------------------------------------------------------------
# Alpine container based off: 
#
# https://github.com/gopheracademy/gopher/blob/master/Dockerfile
# ---------------------------------------------------------------------------------------

MAINTAINER Alex Davies-Moore "a@devork.com"

ENV GRAVAD_GROUP=gravad
ENV GRAVAD_USER=gravad
ENV GRAVAD_HOME=/gravad

RUN echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf && \    
    adduser -h $GRAVAD_HOME -D -s /bin/false $GRAVAD_USER $GRAVAD_GROUP && \
    apk add --update ca-certificates && \
    rm -rf /var/cache/apk/*

ADD gravad /usr/local/bin/gravad
ADD docker/bin/*  /usr/local/bin/

EXPOSE 8080

CMD ["/usr/local/bin/run-gravad.sh"]