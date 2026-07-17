FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
# App binds to $PORT if set (Laravel Cloud injects one), else :9114.
EXPOSE 9114
USER nonroot:nonroot
ENTRYPOINT ["/api"]
