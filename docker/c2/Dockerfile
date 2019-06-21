FROM alpine AS c2

COPY ./bin/c2 /opt/e4/bin/c2
COPY ./configs/ /opt/e4/configs

# Fix glibc dependancy
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

# Redirect log file to stdout
RUN ln -s /dev/stdout /var/log/e4_c2.log

WORKDIR /opt/e4

# Default GRPC
EXPOSE 5555
# Default HTTp
EXPOSE 8888

VOLUME /opt/e4/configs

CMD ["/opt/e4/bin/c2"]
