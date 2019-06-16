#FROM busybox
#COPY  app .
#
#CMD ["./app","--in=true"]

FROM golang:alpine as builder
WORKDIR /go/src/app/
COPY . .
RUN ls
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-d -w -s ' -a -installsuffix cgo -o app .


FROM busybox
WORKDIR /
COPY --from=builder /go/src/app .

CMD ["./app"]