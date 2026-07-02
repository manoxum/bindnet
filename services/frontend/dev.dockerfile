FROM node:22-alpine

WORKDIR /app

RUN apk add --no-cache bash

ENV NODE_ENV=development
ENV PATH=/app/node_modules/.bin:$PATH

EXPOSE 80

CMD ["bash", "bin/entrypoint.dev.sh"]
