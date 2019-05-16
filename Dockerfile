FROM python:3.6-slim

RUN apt-get -y update \
    && apt-get -y install nginx \
    && apt-get -y install python3-dev \
    && apt-get -y install build-essential \
    && apt-get -y install ffmpeg \
    && apt-get clean

RUN pip install gunicorn flask requests pagan youtube-dl

COPY src/main.py /srv/tubefling/main.py
WORKDIR /srv/tubefling/

EXPOSE 80
ENV GUNICORN_TIMEOUT 120
CMD ["gunicorn", "-w", "1", "-b", "0.0.0.0:80", "--timeout", "echo ${GUNICORN_TIMEOUT}", "main:server"]
