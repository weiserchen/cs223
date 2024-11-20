FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download

COPY ./service/ /src/service/
COPY ./pkg/ /src/pkg/
RUN ls -alt
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /bin/service ./service/user/

FROM scratch
COPY --from=build /bin/service /bin/service
ENTRYPOINT [ "/bin/service" ]