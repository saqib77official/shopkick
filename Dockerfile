FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY static static
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /src/server /app/server
COPY --from=build /src/static /app/static
ENV PORT=8080
ENV DB_PATH=/tmp/suggestions.db
EXPOSE 8080
USER nonroot:nonroot
CMD ["/app/server"]
