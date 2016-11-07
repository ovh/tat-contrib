FROM debian:jessie
COPY ./api /app/api
RUN apt-get update && apt-get install -y ca-certificates && chmod +x /app/api && chown -R nobody:nogroup /app/api
USER nobody
CMD ["/app/api"]
