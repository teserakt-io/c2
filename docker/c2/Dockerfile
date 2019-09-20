###########################################################
#
# c2 stage
#
###########################################################
FROM alpine AS c2

ARG binary_path

COPY $binary_path /opt/e4/bin/c2
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
FROM alpine AS c2cli

ARG binary_path

COPY $binary_path /opt/e4/bin/c2cli
COPY ./docker/c2/cli-entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

WORKDIR /opt/e4

VOLUME /opt/c2/c2-cert.pem

ENV C2_API_ENDPOINT="localhost:5555"
ENV C2_API_CERT="/opt/c2/c2-cert.pem"

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
