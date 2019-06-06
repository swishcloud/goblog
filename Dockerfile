FROM gcr.io/intricate-dryad-234705/golangimage@sha256:f449987a095cf3db1778e01219b21244545618056930381ae20ac39774f318f2
RUN go build
CMD ["goblog"]