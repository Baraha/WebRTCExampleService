FROM golang:1.19 as builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build cmd/main.go


FROM scratch
COPY --from=builder /app/main /
COPY --from=builder /app/config.yml /
CMD ["./main"] 
