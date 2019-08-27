FROM scratch
WORKDIR /bin/goblog
COPY . .
ENTRYPOINT ["/bin/goblog/goblog"]
