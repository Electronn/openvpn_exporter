FROM centos:7
COPY openvpn_exporter /usr/bin/
COPY entrypoint.sh /
RUN chmod +x entrypoint.sh && chmod +x /usr/bin/main
EXPOSE 9509:9509
CMD ["/entrypoint.sh"]
