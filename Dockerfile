FROM golang:latest
RUN apt-get update && apt-get install -y ffmpeg
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go build -o main .
CMD ["/app/main"]
