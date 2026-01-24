FROM alpine:3.19

RUN apk add --no-cache \
    pandoc \
    chromium \
    font-noto \
    font-noto-emoji \
    bash

COPY md-to-pdf.sh /usr/local/bin/md-to-pdf
COPY print-style.css /opt/print-style.css

RUN chmod +x /usr/local/bin/md-to-pdf

WORKDIR /data

ENTRYPOINT ["md-to-pdf"]
