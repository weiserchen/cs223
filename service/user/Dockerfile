FROM --platform=$BUILDPLATFORM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download

COPY ./service/ /src/service/
COPY ./pkg/ /src/pkg/

ARG TARGETARCH
ARG TARGETOS
RUN echo "Building User Service image for ${TARGETOS}/${TARGETARCH}..."
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /bin/service ./service/user/

FROM scratch
COPY --from=build /bin/service /bin/service
ENTRYPOINT [ "/bin/service" ]