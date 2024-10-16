FROM golang:bookworm

WORKDIR /mock-collector
COPY ./test/mock/collector/mock-collector/ ./

CMD ["go", "run", "main.go"]
#CMD ["sleep", "infinity"]
