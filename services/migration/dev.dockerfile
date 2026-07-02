FROM node:22-alpine

RUN apk add --no-cache bash openssl

WORKDIR /app

ENV NODE_ENV=development
ENV PATH=/app/node_modules/.bin:$PATH

CMD ["bash", "bin/entrypoint.dev.sh"]
