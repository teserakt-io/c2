###########################################################
#
# c2 stage
#
###########################################################
FROM debian:stable AS c2

COPY ./bin/c2 /opt/e4/bin/c2
COPY ./configs/ /opt/e4/configs

# Redirect log file to stdout
RUN ln -s /dev/stdout /var/log/e4_c2.log

WORKDIR /opt/e4

# Default GRPC
EXPOSE 5555
# Default HTTp
EXPOSE 8888

VOLUME /opt/e4/configs

CMD ["/opt/e4/bin/c2"]

###########################################################
#
# c2cli stage
#
###########################################################
FROM debian:stable AS c2cli

COPY ./docker/c2/cli-entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

WORKDIR /opt/e4

VOLUME /opt/c2/c2-cert.pem

ENV C2_API_ENDPOINT="localhost:5555"
ENV C2_API_CERT="/opt/c2/c2-cert.pem"

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

COPY ./bin/c2cli /opt/e4/bin/c2cli
